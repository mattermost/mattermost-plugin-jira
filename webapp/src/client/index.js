// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ClientError} from '@mattermost/client';

import {Client4} from 'mattermost-redux/client';

export const doFetch = async (url, options) => {
    const {data} = await doFetchWithResponse(url, options);

    return data;
};

export const doFetchWithResponse = async (url, options = {}) => {
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

export function buildQueryString(parameters) {
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
