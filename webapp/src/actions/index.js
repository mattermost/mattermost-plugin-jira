// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import ActionTypes from 'action_types';
import {doFetch} from 'client';
import {getPluginServerRoute} from 'selectors';

export const openCreateModal = (postId) => {
    return {
        type: ActionTypes.OPEN_CREATE_MODAL,
        data: {
            postId,
        },
    };
};

export const closeCreateModal = () => {
    return {
        type: ActionTypes.CLOSE_CREATE_MODAL,
    };
};

export const getCreateIssueMetadata = () => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        let data = null;
        try {
            data = await doFetch(`${baseUrl}/api/v2/get-create-issue-metadata`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_METADATA,
            data,
        });

        return {data};
    };
};

export const createIssue = (payload) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            const data = await doFetch(`${baseUrl}/api/v2/create-issue`, {
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
            data = await doFetch(`${baseUrl}/api/v2/userinfo`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data,
        });

        return {data};
    };
}

export function handleConnectChange(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: msg.data,
        });
    };
}
