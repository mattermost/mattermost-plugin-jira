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
