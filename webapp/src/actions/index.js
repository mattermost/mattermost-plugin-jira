// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {CreateTypes} from 'action_types';
import {doFetch} from 'client';
import {getPluginServerRoute} from 'selectors';

export const openCreateModal = (postId) => {
    return {
        type: CreateTypes.OPEN_CREATE_MODAL,
        data: {
            postId,
        },
    };
};

export const closeCreateModal = () => {
    return {
        type: CreateTypes.CLOSE_CREATE_MODAL,
    };
};

export const getCreateIssueMetadata = () => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            const data = await doFetch(`${baseUrl}/create-issue-metadata`, {
                method: 'get',
            });

            return {data};
        } catch (error) {
            return {error};
        }
    };
};

export const createIssue = (payload) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            const data = await doFetch(`${baseUrl}/create-issue`, {
                method: 'post',
                body: JSON.stringify(payload),
            });

            return {data};
        } catch (error) {
            return {error};
        }
    };
};

export function getConnected() {
    return async (dispatch, getState) => {
        let data;
        const baseUrl = getPluginServerRoute(getState());
        try {
            data = await doFetch(`${baseUrl}/api/v1/connected`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }

        dispatch({
            type: CreateTypes.RECEIVED_CONNECTED,
            data,
        });

        return {data};
    };
}

export function handleConnect(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: CreateTypes.RECEIVED_CONNECTED,
            data: msg.data,
        });
    };
}