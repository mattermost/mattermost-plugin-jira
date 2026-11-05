// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-console */

import {createHash, randomBytes} from 'node:crypto';
import {createServer, IncomingMessage, Server, ServerResponse} from 'node:http';

import {ContractValidator} from './contract/validator';
import {
    assignableUsers,
    cloudUser,
    CloudSite,
    createIssue,
    CreateIssueFields,
    createMeta,
    defaultState,
    FakeJiraState,
    issueByKey,
    issueComment,
    issueTransitions,
    JournalEntry,
    projectByKey,
    projectSearch,
    projectStatuses,
    searchIssues,
    serverInfo,
    serverUser,
} from './state';

type RequestContext = {
    req: IncomingMessage;
    res: ServerResponse;
    url: URL;
    body: string;
    state: FakeJiraState;

    // Set when the request came through Atlassian's API gateway (`/ex/jira/{cloudId}`).
    cloud: boolean;
};

type Handler = (ctx: RequestContext) => void;

// Paths are matched after the `/ex/jira/{cloudId}` gateway prefix is stripped, so v2/v3 REST
// handlers serve both flavors; `ctx.cloud` tells them which shape to return.
const routes: Record<string, Handler> = {
    'GET /status': ({res}) => sendJSON(res, 200, {state: 'RUNNING'}),

    // Jira Server / Data Center: OAuth 1.0a
    'POST /plugins/servlet/oauth/request-token': ({req, res, state}) => {
        const oauth = parseOAuthHeader(req.headers.authorization);
        const token = `req-${randomHex()}`;
        state.requestTokens.set(token, {callback: oauth.oauth_callback ?? '', verifier: randomHex()});
        sendForm(res, {oauth_token: token, oauth_token_secret: randomHex(), oauth_callback_confirmed: 'true'});
    },

    'GET /plugins/servlet/oauth/authorize': ({res, url, state}) => {
        const token = url.searchParams.get('oauth_token') ?? '';
        const pending = state.requestTokens.get(token);
        if (!pending?.callback) {
            sendJSON(res, 400, {errorMessages: [`unknown request token ${token}`]});
            return;
        }
        redirect(res, pending.callback, {oauth_token: token, oauth_verifier: pending.verifier});
    },

    'POST /plugins/servlet/oauth/access-token': ({req, res, state}) => {
        const oauth = parseOAuthHeader(req.headers.authorization);
        const pending = state.requestTokens.get(oauth.oauth_token ?? '');
        if (!pending || pending.verifier !== oauth.oauth_verifier) {
            sendJSON(res, 401, {errorMessages: ['invalid request token or verifier']});
            return;
        }
        state.requestTokens.delete(oauth.oauth_token!);
        sendForm(res, {oauth_token: issueAccessToken(state), oauth_token_secret: randomHex()});
    },

    // Jira Cloud: OAuth 2.0 (3LO) with PKCE, served on auth.atlassian.com / api.atlassian.com
    'GET /authorize': ({res, url, state}) => {
        const params = url.searchParams;
        if (params.get('client_id') !== state.cloud.clientId) {
            sendJSON(res, 400, {error: 'invalid_client', error_description: `unknown client_id ${params.get('client_id')}`});
            return;
        }
        if (params.get('code_challenge_method') !== 'S256' || !params.get('code_challenge')) {
            sendJSON(res, 400, {error: 'invalid_request', error_description: 'PKCE S256 code_challenge is required'});
            return;
        }
        const code = `code-${randomHex()}`;
        state.authorizationCodes.set(code, {redirectURI: params.get('redirect_uri') ?? '', codeChallenge: params.get('code_challenge')!});
        redirect(res, params.get('redirect_uri') ?? '', {code, state: params.get('state') ?? ''});
    },

    'POST /oauth/token': ({req, res, body, state}) => {
        const form = new URLSearchParams(body);
        const [clientId, clientSecret] = clientCredentials(req, form);
        if (clientId !== state.cloud.clientId || clientSecret !== state.cloud.clientSecret) {
            sendJSON(res, 401, {error: 'access_denied', error_description: 'Unauthorized'});
            return;
        }

        if (form.get('grant_type') === 'refresh_token') {
            if (!state.refreshTokens.delete(form.get('refresh_token') ?? '')) {
                sendJSON(res, 403, {error: 'invalid_grant', error_description: 'Unknown or invalid refresh token.'});
                return;
            }
            sendJSON(res, 200, issueOAuth2Token(state));
            return;
        }

        const pending = state.authorizationCodes.get(form.get('code') ?? '');
        const verifier = form.get('code_verifier') ?? '';
        if (!pending || pending.redirectURI !== form.get('redirect_uri') || s256(verifier) !== pending.codeChallenge) {
            sendJSON(res, 403, {error: 'invalid_grant', error_description: 'Invalid authorization code'});
            return;
        }
        state.authorizationCodes.delete(form.get('code')!);
        sendJSON(res, 200, issueOAuth2Token(state));
    },

    'GET /oauth/token/accessible-resources': ({req, res, state}) => {
        if (!isAuthorized(req, state)) {
            sendJSON(res, 401, {code: 401, message: 'Unauthorized'});
            return;
        }
        sendJSON(res, 200, [{id: state.cloud.cloudId, url: state.cloud.siteURL, name: 'e2e', scopes: [], avatarUrl: ''}]);
    },

    // REST, shared by both flavors
    'GET /rest/api/2/myself': myself,
    'GET /rest/api/3/myself': myself,
    'GET /rest/api/2/serverInfo': ({res, state}) => sendJSON(res, 200, serverInfo(state.baseURL)),

    // Jira Cloud project data behind the create-issue, attach-comment and subscription modals
    'GET /rest/api/3/project/search': authorized(({res, url, state}) => {
        sendJSON(res, 200, projectSearch(state.cloud, url.searchParams.get('expand')?.includes('issueTypes') ?? false));
    }),
    'GET /rest/api/2/project/{projectIdOrKey}': authorized(({res, url, state}) => {
        const key = decodeURIComponent(url.pathname.split('/').pop() ?? '');
        const project = projectByKey(state.cloud, key);
        if (!project) {
            sendJSON(res, 404, {errorMessages: [`No project could be found with key '${key}'.`], errors: {}});
            return;
        }
        sendJSON(res, 200, project);
    }),
    'GET /rest/api/3/project/{projectIdOrKey}/statuses': authorized(({res, state}) => sendJSON(res, 200, projectStatuses(state.cloud))),
    'GET /rest/api/2/issue/createmeta': authorized(({res, state}) => sendJSON(res, 200, createMeta(state.cloud))),
    'GET /rest/api/2/search/jql': authorized(({res, state}) => sendJSON(res, 200, searchIssues(state.cloud))),
    'GET /rest/api/2/issue/{issueIdOrKey}': authorized(({res, url, state}) => {
        const key = decodeURIComponent(url.pathname.split('/').pop() ?? '');
        const issue = issueByKey(state.cloud, key, state.createdIssues);
        if (!issue) {
            sendJSON(res, 404, {errorMessages: ['Issue does not exist or you do not have permission to see it.'], errors: {}});
            return;
        }
        sendJSON(res, 200, issue);
    }),
    'GET /rest/api/2/issue/{issueIdOrKey}/transitions': authorized(({res, state}) => sendJSON(res, 200, issueTransitions(state.cloud))),
    'POST /rest/api/2/issue': authorized(({res, body, state}) => {
        const {fields = {}} = JSON.parse(body) as {fields?: CreateIssueFields};
        if (!fields.summary) {
            sendJSON(res, 400, {errorMessages: [], errors: {summary: 'You must specify a summary of the issue.'}});
            return;
        }
        sendJSON(res, 201, createIssue(state, fields));
    }),
    'POST /rest/api/2/issue/{issueIdOrKey}/comment': authorized(withIssue(({res, body, state}, issue) => {
        sendJSON(res, 201, issueComment(state.cloud, issue.id, (JSON.parse(body) as {body?: string}).body ?? ''));
    })),
    'POST /rest/api/2/issue/{issueIdOrKey}/transitions': authorized(withIssue(({res, body, state}) => {
        const id = (JSON.parse(body) as {transition?: {id?: string}}).transition?.id;
        if (!issueTransitions(state.cloud).transitions.some((t) => t.id === id)) {
            sendJSON(res, 400, {errorMessages: [`Transition id '${id}' is not valid for this issue.`], errors: {}});
            return;
        }
        res.writeHead(204).end();
    })),
    'PUT /rest/api/2/issue/{issueIdOrKey}/assignee': authorized(withIssue(({res}) => res.writeHead(204).end())),
    'GET /rest/api/2/user/assignable/search': authorized(({res, url, state}) => {
        sendJSON(res, 200, assignableUsers(state.cloud, url.searchParams.get('query') ?? ''));
    }),
};

const templateRoutes = Object.entries(routes).
    filter(([key]) => key.includes('{')).
    map(([key, handler]) => ({pattern: new RegExp(`^${key.replace(/\{[^}]+\}/g, '[^/]+')}$`), handler}));

function findRoute(key: string): Handler | undefined {
    return routes[key] ?? templateRoutes.find(({pattern}) => pattern.test(key))?.handler;
}

function authorized(handler: Handler): Handler {
    return (ctx) => {
        if (!isAuthorized(ctx.req, ctx.state)) {
            sendJSON(ctx.res, 401, {errorMessages: ['not authenticated']});
            return;
        }
        handler(ctx);
    };
}

function withIssue(handler: (ctx: RequestContext, issue: NonNullable<ReturnType<typeof issueByKey>>) => void): Handler {
    return (ctx) => {
        const key = decodeURIComponent(ctx.url.pathname.match(/\/issue\/([^/]+)\//)?.[1] ?? '');
        const issue = issueByKey(ctx.state.cloud, key, ctx.state.createdIssues);
        if (!issue) {
            sendJSON(ctx.res, 404, {errorMessages: ['Issue does not exist or you do not have permission to see it.'], errors: {}});
            return;
        }
        handler(ctx, issue);
    };
}

function myself({req, res, state, cloud}: RequestContext) {
    if (!isAuthorized(req, state)) {
        sendJSON(res, 401, {errorMessages: ['not authenticated']});
        return;
    }
    sendJSON(res, 200, cloud ? cloudUser(state.cloud) : serverUser(state.baseURL));
}

const controlRoutes: Record<string, Handler> = {
    'POST /__control/reset': ({res, state}) => {
        Object.assign(state, defaultState(state.baseURL, state.cloud));
        sendJSON(res, 200, {ok: true});
    },
    'GET /__control/requests': ({res, state}) => sendJSON(res, 200, state.journal),
    'GET /__control/contract-violations': ({res, state}) => sendJSON(res, 200, state.contractViolations),
    'POST /__control/tokens': ({res, state}) => sendJSON(res, 200, issueOAuth2Token(state)),
    'POST /__control/fault': ({res, body, state}) => {
        const {method, path, status} = JSON.parse(body) as {method: string; path: string; status: number};
        state.faults.set(`${method.toUpperCase()} ${path}`, status);
        sendJSON(res, 200, {ok: true});
    },
};

export type FakeJira = {
    server: Server;
    close: () => Promise<void>;
};

export function startFakeJira({port, baseURL, cloud}: {port: number; baseURL: string; cloud: CloudSite}): Promise<FakeJira> {
    const state = defaultState(baseURL, cloud);
    const gatewayPrefix = `/ex/jira/${cloud.cloudId}`;
    const contract = new ContractValidator();

    const handleRequest = async (req: IncomingMessage, res: ServerResponse) => {
        const url = new URL(req.url ?? '/', 'http://fake-jira');
        const body = await readBody(req);
        const isCloud = url.pathname.startsWith(gatewayPrefix);
        const path = isCloud ? url.pathname.slice(gatewayPrefix.length) : url.pathname;
        const key = `${req.method} ${path}`;
        const ctx: RequestContext = {req, res, url, body, state, cloud: isCloud};

        const control = controlRoutes[key];
        if (control) {
            control(ctx);
            return;
        }

        const entry = toJournalEntry(req, url, body);
        state.journal.push(entry);

        const fault = state.faults.get(key);
        if (fault) {
            sendJSON(res, fault, {errorMessages: [`fault injected for ${key}`]});
            return;
        }

        if (isCloud) {
            state.contractViolations.push(...contract.validateRequest(req.method ?? '', path, entry.body));
            checkResponseContract(res, (status, payload) => {
                state.contractViolations.push(...contract.validateResponse(req.method ?? '', path, status, payload));
            });
        }

        const handler = findRoute(key);
        if (!handler) {
            console.warn(`[fake-jira] unhandled ${req.method} ${url.pathname}`);
            sendJSON(res, 404, {errorMessages: [`fake-jira has no route for ${key}`]});
            return;
        }
        handler(ctx);
    };

    // A throw in a handler would otherwise be an unhandled rejection, which kills the fake for the rest of the run.
    const server = createServer((req, res) => {
        handleRequest(req, res).catch((err) => {
            console.error(`[fake-jira] ${req.method} ${req.url} failed:`, err);
            if (res.headersSent) {
                res.end();
                return;
            }
            sendJSON(res, 500, {errorMessages: [`fake-jira error: ${err instanceof Error ? err.message : String(err)}`]});
        });
    });

    return new Promise((resolve) => {
        server.listen(port, () => {
            console.log(`[fake-jira] listening on :${port} as ${baseURL} and ${cloud.siteURL}`);
            resolve({
                server,
                close: () => new Promise((done) => server.close(() => done())),
            });
        });
    });
}

function checkResponseContract(res: ServerResponse, check: (status: number, payload: unknown) => void) {
    const end = res.end.bind(res);
    res.end = ((chunk?: unknown, ...rest: unknown[]) => {
        if (typeof chunk === 'string') {
            try {
                check(res.statusCode, JSON.parse(chunk));
            } catch {
                // non-JSON responses have no schema to check
            }
        }
        return (end as (...args: unknown[]) => ServerResponse)(chunk, ...rest);
    }) as typeof res.end;
}

function issueAccessToken(state: FakeJiraState) {
    const token = `acc-${randomHex()}`;
    state.accessTokens.add(token);
    return token;
}

function issueOAuth2Token(state: FakeJiraState) {
    const refreshToken = `ref-${randomHex()}`;
    state.refreshTokens.add(refreshToken);
    return {
        access_token: issueAccessToken(state),
        refresh_token: refreshToken,
        expires_in: 3600,
        token_type: 'Bearer',
        scope: 'read:jira-user read:jira-work write:jira-work offline_access',
    };
}

function isAuthorized(req: IncomingMessage, state: FakeJiraState) {
    const header = req.headers.authorization ?? '';
    const token = header.startsWith('Bearer ') ? header.slice('Bearer '.length) : parseOAuthHeader(header).oauth_token;
    return Boolean(token && state.accessTokens.has(token));
}

// golang.org/x/oauth2 auto-detects the auth style: Basic auth header first, then form fields.
function clientCredentials(req: IncomingMessage, form: URLSearchParams): [string, string] {
    const header = req.headers.authorization ?? '';
    if (header.startsWith('Basic ')) {
        const [id, secret] = Buffer.from(header.slice('Basic '.length), 'base64').toString().split(':');
        return [decodeURIComponent(id), decodeURIComponent(secret ?? '')];
    }
    return [form.get('client_id') ?? '', form.get('client_secret') ?? ''];
}

function parseOAuthHeader(header?: string): Record<string, string> {
    const params: Record<string, string> = {};
    if (!header?.startsWith('OAuth ')) {
        return params;
    }
    for (const [, key, value] of header.matchAll(/(\w+)="([^"]*)"/g)) {
        params[key] = decodeURIComponent(value);
    }
    return params;
}

function s256(verifier: string) {
    return createHash('sha256').update(verifier).digest('base64url');
}

function toJournalEntry(req: IncomingMessage, url: URL, body: string): JournalEntry {
    let parsed: unknown = body;
    try {
        parsed = body ? JSON.parse(body) : undefined;
    } catch {
        // non-JSON bodies are journaled as raw text
    }
    return {method: req.method ?? '', path: url.pathname, query: Object.fromEntries(url.searchParams), body: parsed};
}

function readBody(req: IncomingMessage): Promise<string> {
    return new Promise((resolve, reject) => {
        let data = '';
        req.on('data', (chunk) => {
            data += chunk;
        });
        req.on('end', () => resolve(data));
        req.on('error', reject);
    });
}

function redirect(res: ServerResponse, target: string, params: Record<string, string>) {
    const location = new URL(target);
    for (const [key, value] of Object.entries(params)) {
        location.searchParams.set(key, value);
    }
    res.writeHead(302, {Location: location.toString()}).end();
}

function sendJSON(res: ServerResponse, status: number, payload: unknown) {
    res.writeHead(status, {'Content-Type': 'application/json'}).end(JSON.stringify(payload));
}

function sendForm(res: ServerResponse, payload: Record<string, string>) {
    res.writeHead(200, {'Content-Type': 'application/x-www-form-urlencoded'}).end(new URLSearchParams(payload).toString());
}

function randomHex() {
    return randomBytes(8).toString('hex');
}
