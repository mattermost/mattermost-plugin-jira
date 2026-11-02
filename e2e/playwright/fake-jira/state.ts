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
};

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
    {id: '10100', key: 'E2E-1', summary: 'Checkout button overlaps the cart total on narrow screens'},
    {id: '10101', key: 'E2E-2', summary: 'Nightly export fails for workspaces with archived channels'},
];

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

export function projectSearch(site: CloudSite, expandIssueTypes: boolean) {
    const project = {
        self: `${site.siteURL}/rest/api/3/project/${fakeProject.id}`,
        ...fakeProject,
        projectTypeKey: 'software',
        simplified: false,
        ...(expandIssueTypes ? {issueTypes: issueTypes.map((t) => ({self: `${site.siteURL}/rest/api/3/issuetype/${t.id}`, ...t, subtask: false}))} : {}),
    };
    return {self: `${site.siteURL}/rest/api/3/project/search`, startAt: 0, maxResults: 50, total: 1, isLast: true, values: [project]};
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
    };
}
