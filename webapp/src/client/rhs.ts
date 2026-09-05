// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from 'mattermost-redux/client';

import {
    RHSErrorCode,
    RHSIssuesResponse,
    RHSSort,
    RHSStatusesResponse,
    RHSTab,
    rhsTabFromAPI,
    rhsTabIdentity,
} from 'types/rhs';

import {buildQueryString} from './index';

interface QueryParameters {
    [key: string]: string | number | boolean;
}

interface FetchOptions {
    method: string;
    body?: BodyInit | null;
}

export class RHSFetchError extends Error {
    errorCode: RHSErrorCode;
    statusCode: number;

    constructor(errorCode: RHSErrorCode, message: string, statusCode: number) {
        super(message);
        this.name = 'RHSFetchError';
        this.errorCode = errorCode;
        this.statusCode = statusCode;
    }
}

export function coerceRHSErrorCode(value: string): RHSErrorCode {
    switch (value) {
    case 'not_connected':
    case 'rate_limited':
    case 'not_authorized':
    case 'not_cloud':
    case 'invalid_request':
    case 'internal_error':
        return value;
    default:
        return 'internal_error';
    }
}

export function parseRHSErrorBody(text: string, statusCode: number): RHSFetchError {
    try {
        const parsed = JSON.parse(text);
        if (
            parsed &&
            typeof parsed === 'object' &&
            typeof parsed.error === 'string'
        ) {
            const message = typeof parsed.message === 'string' && parsed.message ?
                parsed.message :
                parsed.error;
            return new RHSFetchError(coerceRHSErrorCode(parsed.error), message, statusCode);
        }
    } catch {
        return new RHSFetchError('internal_error', text || '', statusCode);
    }

    return new RHSFetchError('internal_error', text || '', statusCode);
}

const doFetchJSON = async <T>(url: string, options: FetchOptions): Promise<T> => {
    const response = await fetch(url, Client4.getOptions(options));
    if (response.ok) {
        return response.json();
    }

    const text = await response.text();
    throw parseRHSErrorBody(text, response.status);
};

export type GetRHSIssuesParams = {
    instanceID: string;
    tab: RHSTab;
    sort: RHSSort;
    nextPageToken?: string;
};

function buildRHSIssuesQuery(params: GetRHSIssuesParams): QueryParameters {
    const query: QueryParameters = {
        instance_id: params.instanceID,
        sort: params.sort,
        tab: rhsTabIdentity(params.tab),
    };
    if (params.nextPageToken) {
        query.next_page_token = params.nextPageToken;
    }

    return query;
}

function parseRHSTabs(raw: unknown): RHSTab[] {
    if (!Array.isArray(raw)) {
        return [];
    }

    const tabs: RHSTab[] = [];
    for (let i = 0; i < raw.length; i++) {
        const item = raw[i];
        if (!item || typeof item !== 'object') {
            continue;
        }
        const tab = rhsTabFromAPI(item);
        if (tab) {
            tabs.push(tab);
        }
    }
    return tabs;
}

function parseRHSIssuesResponse(raw: RHSIssuesResponse): RHSIssuesResponse {
    return {
        issues: raw.issues || [],
        tabs: parseRHSTabs(raw.tabs),
        nextPageToken: raw.nextPageToken || '',
        isLast: Boolean(raw.isLast),
    };
}

export async function getRHSIssues(baseUrl: string, params: GetRHSIssuesParams): Promise<RHSIssuesResponse> {
    const raw = await doFetchJSON<RHSIssuesResponse>(
        `${baseUrl}/api/v2/rhs/issues${buildQueryString(buildRHSIssuesQuery(params))}`,
        {method: 'get'},
    );
    return parseRHSIssuesResponse(raw);
}

export function getRHSStatuses(baseUrl: string, instanceID: string): Promise<RHSStatusesResponse> {
    return doFetchJSON<RHSStatusesResponse>(
        `${baseUrl}/api/v2/rhs/statuses${buildQueryString({instance_id: instanceID})}`,
        {method: 'get'},
    );
}
