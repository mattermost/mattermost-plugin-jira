// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export type RHSErrorCode =
    'not_connected' |
    'rate_limited' |
    'not_authorized' |
    'not_cloud' |
    'invalid_request' |
    'internal_error';

export type RHSSort = 'updated' | 'created';

export type RHSTab =
    | {kind: 'assigned'; name: string}
    | {kind: 'category'; key: string; name: string}
    | {kind: 'status'; id: string; name: string};

export type RHSIssueStatus = {
    name: string;
    categoryKey: string;
};

export type RHSIssue = {
    key: string;
    summary: string;
    browseUrl: string;
    status: RHSIssueStatus;
    issueType: string;
    issueTypeIconUrl?: string;
    project: string;
    updated: string;
};

export type RHSIssuesResponse = {
    issues: RHSIssue[];
    tabs: RHSTab[];
    nextPageToken: string;
    isLast: boolean;
};

export type RHSStatusCategory = {
    id: number;
    key: string;
    name: string;
};

export type RHSStatusProject = {
    id?: string;
    key?: string;
    name?: string;
};

export type RHSStatus = {
    id: string;
    name: string;
    statusCategory: RHSStatusCategory;
    project?: RHSStatusProject;
};

export type RHSStatusesResponse = {
    statuses: RHSStatus[];
    categories: RHSStatusCategory[];
};

export type RHSViewState = {
    instance: string;
    tab: RHSTab;
    sort: RHSSort;
};

export const RHS_DEFAULT_SORT: RHSSort = 'updated';

export const RHS_ASSIGNED_TAB_NAME = 'Assigned to me';

export const RHS_DEFAULT_TAB: RHSTab = {
    kind: 'assigned',
    name: RHS_ASSIGNED_TAB_NAME,
};

export type FetchRHSIssuesArgs = {
    instanceID: string;
    tab: RHSTab;
    sort: RHSSort;
};

export type GetRHSIssuesParams = FetchRHSIssuesArgs & {
    nextPageToken?: string;
};

export function rhsTabIdentity(tab: RHSTab): string {
    switch (tab.kind) {
    case 'assigned':
        return 'assigned';
    case 'category':
        return 'category:' + tab.key;
    case 'status':
        return 'status:' + tab.id;
    default: {
        const exhaustive: never = tab;
        return exhaustive;
    }
    }
}

export function rhsTabFromAPI(raw: {kind?: unknown; name?: unknown; key?: unknown; id?: unknown}): RHSTab | null {
    const name = typeof raw.name === 'string' && raw.name ? raw.name : '';
    switch (raw.kind) {
    case 'assigned':
        return {
            kind: 'assigned',
            name: RHS_ASSIGNED_TAB_NAME,
        };
    case 'category': {
        if (typeof raw.key !== 'string' || !raw.key) {
            return null;
        }
        return {
            kind: 'category',
            key: raw.key,
            name: name || raw.key,
        };
    }
    case 'status': {
        if (typeof raw.id !== 'string' || !raw.id) {
            return null;
        }
        return {
            kind: 'status',
            id: raw.id,
            name: name || raw.id,
        };
    }
    default:
        return null;
    }
}
