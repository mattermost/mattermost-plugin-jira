// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export type JournalEntry = {
    method: string;
    path: string;
    query: Record<string, string>;
    body: unknown;
};

export type CloudSite = {
    cloudId: string;
    siteURL: string;
    clientId: string;
    clientSecret: string;
};

type PendingAuthorization = {
    redirectURI: string;
    codeChallenge: string;
};

export type FakeJiraState = {
    baseURL: string;
    cloud: CloudSite;
    requestTokens: Map<string, {callback: string; verifier: string}>;
    authorizationCodes: Map<string, PendingAuthorization>;
    refreshTokens: Set<string>;
    accessTokens: Set<string>;
    journal: JournalEntry[];
    faults: Map<string, number>;
    contractViolations: string[];
    createdIssues: CreatedIssue[];
};

export type CreatedIssue = {id: string; key: string; summary: string; description: string; labels: string[]; issueTypeId: string};

export type CreateIssueFields = {summary?: string; description?: string; labels?: string[]; issuetype?: {id?: string}};

export const fakeJiraUser = {
    key: 'e2e-jira-user',
    name: 'e2e-jira-user',
    displayName: 'E2E Jira User',
    emailAddress: 'e2e-jira-user@example.com',
    active: true,
    timeZone: 'UTC',
};

export const fakeCloudUser = {
    accountId: '5f0000000000000000e2e001',
    accountType: 'atlassian',
    displayName: 'E2E Cloud User',
    emailAddress: 'e2e-cloud-user@example.com',
    active: true,
    timeZone: 'UTC',
};

export function serverUser(baseURL: string) {
    return {self: `${baseURL}/rest/api/2/user?username=${fakeJiraUser.name}`, ...fakeJiraUser};
}

export function cloudUser(site: CloudSite) {
    return {self: `${site.siteURL}/rest/api/3/user?accountId=${fakeCloudUser.accountId}`, ...fakeCloudUser};
}

export const fakeProject = {id: '10000', key: 'E2E', name: 'E2E Project'};

const issueTypes = [
    {id: '10001', name: 'Task', description: 'A small, distinct piece of work.'},
    {id: '10002', name: 'Bug', description: 'A problem or error.'},
];

const statuses = [
    {id: '1', name: 'To Do'},
    {id: '3', name: 'In Progress'},
    {id: '10001', name: 'Done'},
];

export const fakeIssues = [
    {
        id: '10100',
        key: 'E2E-1',
        summary: 'Checkout button overlaps the cart total on narrow screens',
        description: 'On a 375px viewport the *Checkout* button covers the cart total, so shoppers cannot see what they pay.',
        labels: ['checkout', 'mobile'],
    },
    {
        id: '10101',
        key: 'E2E-2',
        summary: 'Nightly export fails for workspaces with archived channels',
        description: 'The export job stops at the first archived channel and never uploads the archive.',
        labels: ['export'],
    },
];

// Inline, so the dockerized browser renders it without reaching fake-jira.
const issueTypeIcon = `data:image/svg+xml,${encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><rect width="16" height="16" rx="3" fill="#e5493a"/></svg>')}`;

export const createFields = {
    summary: {required: true, schema: {type: 'string', system: 'summary'}, name: 'Summary', key: 'summary', hasDefaultValue: false, operations: ['set']},
    description: {required: false, schema: {type: 'string', system: 'description'}, name: 'Description', key: 'description', hasDefaultValue: false, operations: ['set']},
    priority: {
        required: false,
        schema: {type: 'priority', system: 'priority'},
        name: 'Priority',
        key: 'priority',
        hasDefaultValue: true,
        operations: ['set'],
        allowedValues: ['Highest', 'High', 'Medium', 'Low'].map((name, i) => ({id: String(i + 1), name})),
        defaultValue: {id: '3', name: 'Medium'},
    },
};

function project(site: CloudSite, apiVersion: 2 | 3, expandIssueTypes: boolean) {
    const api = `${site.siteURL}/rest/api/${apiVersion}`;
    return {
        self: `${api}/project/${fakeProject.id}`,
        ...fakeProject,
        projectTypeKey: 'software',
        simplified: false,
        ...(expandIssueTypes ? {issueTypes: issueTypes.map((t) => ({self: `${api}/issuetype/${t.id}`, ...t, subtask: false}))} : {}),
    };
}

export function projectSearch(site: CloudSite, expandIssueTypes: boolean) {
    return {self: `${site.siteURL}/rest/api/3/project/search`, startAt: 0, maxResults: 50, total: 1, isLast: true, values: [project(site, 3, expandIssueTypes)]};
}

export function projectByKey(site: CloudSite, idOrKey: string) {
    return idOrKey === fakeProject.key || idOrKey === fakeProject.id ? project(site, 2, true) : undefined;
}

export function projectStatuses(site: CloudSite) {
    return issueTypes.map((t) => ({
        self: `${site.siteURL}/rest/api/3/issuetype/${t.id}`,
        id: t.id,
        name: t.name,
        subtask: false,
        statuses: statuses.map((s) => ({self: `${site.siteURL}/rest/api/3/status/${s.id}`, ...s})),
    }));
}

export function createMeta(site: CloudSite, fields: Partial<typeof createFields> = createFields) {
    return {
        projects: [{
            self: `${site.siteURL}/rest/api/2/project/${fakeProject.id}`,
            ...fakeProject,
            issuetypes: issueTypes.map((t) => ({self: `${site.siteURL}/rest/api/2/issuetype/${t.id}`, ...t, subtask: false, fields})),
        }],
    };
}

export function searchIssues(site: CloudSite) {
    return {
        isLast: true,
        issues: fakeIssues.map(({id, key, summary}) => ({id, key, self: `${site.siteURL}/rest/api/2/issue/${id}`, fields: {summary}})),
    };
}

export function issueByKey(site: CloudSite, idOrKey: string, created: CreatedIssue[] = []) {
    const issues: Array<(typeof fakeIssues)[number] & {issueTypeId?: string}> = [...fakeIssues, ...created];
    const issue = issues.find((i) => i.key === idOrKey || i.id === idOrKey);
    if (!issue) {
        return undefined;
    }
    const api = `${site.siteURL}/rest/api/2`;
    const bug = issueTypes.find((t) => t.id === issue.issueTypeId) ?? issueTypes[1];
    return {
        id: issue.id,
        key: issue.key,
        self: `${api}/issue/${issue.id}`,
        fields: {
            summary: issue.summary,
            description: issue.description,
            labels: issue.labels,
            issuetype: {self: `${api}/issuetype/${bug.id}`, ...bug, iconUrl: issueTypeIcon, subtask: false},
            project: {self: `${api}/project/${fakeProject.id}`, ...fakeProject},
            status: {self: `${api}/status/${statuses[0].id}`, ...statuses[0]},
            priority: {self: `${api}/priority/2`, id: '2', name: 'High'},
            assignee: cloudUser(site),
        },
    };
}

export function issueTransitions(site: CloudSite) {
    return {
        transitions: statuses.map((s) => ({id: s.id, name: s.name, to: {self: `${site.siteURL}/rest/api/2/status/${s.id}`, ...s}})),
    };
}

export function createIssue(state: FakeJiraState, fields: CreateIssueFields) {
    const id = String(10200 + state.createdIssues.length);
    const key = `${fakeProject.key}-${fakeIssues.length + state.createdIssues.length + 1}`;
    state.createdIssues.push({
        id,
        key,
        summary: fields.summary ?? '',
        description: fields.description ?? '',
        labels: fields.labels ?? [],
        issueTypeId: fields.issuetype?.id ?? issueTypes[0].id,
    });
    return {id, key, self: `${state.cloud.siteURL}/rest/api/2/issue/${id}`};
}

export function issueComment(site: CloudSite, issueId: string, body: string) {
    const now = new Date().toISOString();
    const author = cloudUser(site);
    return {self: `${site.siteURL}/rest/api/2/issue/${issueId}/comment/20000`, id: '20000', author, updateAuthor: author, body, created: now, updated: now};
}

// Cloud only shows other users' email addresses when their profile visibility allows it, so it's left out.
export function assignableUsers(site: CloudSite, query: string) {
    const {emailAddress, ...user} = cloudUser(site);
    return [user].filter((u) => u.displayName.toLowerCase().includes(query.toLowerCase()) || emailAddress.startsWith(query.toLowerCase()));
}

export function serverInfo(baseURL: string) {
    return {baseUrl: baseURL, version: '9.12.0', versionNumbers: [9, 12, 0], deploymentType: 'Server', serverTitle: 'Fake Jira'};
}

export function defaultState(baseURL: string, cloud: CloudSite): FakeJiraState {
    return {
        baseURL,
        cloud,
        requestTokens: new Map(),
        authorizationCodes: new Map(),
        refreshTokens: new Set(),
        accessTokens: new Set(),
        journal: [],
        faults: new Map(),
        contractViolations: [],
        createdIssues: [],
    };
}
