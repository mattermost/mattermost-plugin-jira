// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4} from 'mattermost-redux/client';

export const doFetch = async (url, options) => {
    const {data} = await doFetchWithResponse(url, options);

    return data;
};

export const doFetchWithResponse = async (url, options = {}) => {
    const response = await fetch(url, Client4.getOptions(options));

    let data;
    if (response.ok) {
        try {
            data = await response.json();
        } catch (err) {
            throw err;
        }

        return {
            response,
            data,
        };
    }

    data = await response.text();

    throw new Error(data);
};
