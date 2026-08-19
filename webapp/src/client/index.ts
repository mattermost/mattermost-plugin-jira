// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from 'mattermost-redux/client';
import {ClientError} from '@mattermost/client';

import {
    RHSErrorCode,
    RHSIssuesResponse,
    RHSSort,
    RHSStatusesResponse,
    RHSTab,
} from 'types/model';

interface QueryParameters {
    [key: string]: string | number | boolean;
}

interface FetchOptions {
    method: string;
    body?: BodyInit | null;
}

export const doFetch = async (url: string, options: FetchOptions) => {
    const {data} = await doFetchWithResponse(url, options);

    return data;
};

export const doFetchWithResponse = async (url: string, options = {}) => {
    const response = await fetch(url, Client4.getOptions(options));

    let data;
    if (response.ok) {
        data = await response.json();

        return {
            response,
            data,
        };
    }

    data = await response.text();

    throw new ClientError(Client4.url, {
        message: data || '',
        status_code: response.status,
        url,
    });
};

export function buildQueryString(parameters: QueryParameters) {
    const keys = Object.keys(parameters);
    if (keys.length === 0) {
        return '';
    }

    let query = '?';
    for (let i = 0; i < keys.length; i++) {
        const key = keys[i];
        query += key + '=' + encodeURIComponent(parameters[key]);

        if (i < keys.length - 1) {
            query += '&';
        }
    }

    return query;
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

export const doFetchJSON = async <T>(url: string, options: FetchOptions): Promise<T> => {
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
        tab_kind: params.tab.kind,
    };
    if (params.tab.key) {
        query.tab_key = params.tab.key;
    }
    if (params.tab.id) {
        query.tab_id = params.tab.id;
    }
    if (params.nextPageToken) {
        query.next_page_token = params.nextPageToken;
    }

    return query;
}

export function getRHSIssues(baseUrl: string, params: GetRHSIssuesParams): Promise<RHSIssuesResponse> {
    return doFetchJSON<RHSIssuesResponse>(
        `${baseUrl}/api/v2/rhs/issues${buildQueryString(buildRHSIssuesQuery(params))}`,
        {method: 'get'},
    );
}

export function getRHSStatuses(baseUrl: string, instanceID: string): Promise<RHSStatusesResponse> {
    return doFetchJSON<RHSStatusesResponse>(
        `${baseUrl}/api/v2/rhs/statuses${buildQueryString({instance_id: instanceID})}`,
        {method: 'get'},
    );
}
