// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PostTypes} from 'mattermost-redux/action_types';
import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import ActionTypes from 'action_types';
import {doFetch} from 'client';
import {getPluginServerRoute} from 'selectors';

export const openCreateModal = (postId) => {
    return {
        type: ActionTypes.OPEN_CREATE_ISSUE_MODAL,
        data: {
            postId,
        },
    };
};

export const openCreateModalWithoutPost = (description, channelId) => (dispatch) => dispatch({
    type: ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST,
    data: {
        description,
        channelId,
    },
});

export const closeCreateModal = () => {
    return {
        type: ActionTypes.CLOSE_CREATE_ISSUE_MODAL,
    };
};

export const openAttachCommentToIssueModal = (postId) => {
    return {
        type: ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL,
        data: {
            postId,
        },
    };
};

export const closeAttachCommentToIssueModal = () => {
    return {
        type: ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL,
    };
};

export const fetchJiraIssueMetadata = () => {
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
            type: ActionTypes.RECEIVED_JIRA_ISSUE_METADATA,
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
export const attachCommentToIssue = (payload) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            const data = await doFetch(`${baseUrl}/api/v2/attach-comment-to-issue`, {
                method: 'post',
                body: JSON.stringify(payload),
            });

            return {data};
        } catch (error) {
            return {error};
        }
    };
};

export const createChannelSubscription = (subscription) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            const data = await doFetch(`${baseUrl}/api/v2/subscriptions/channel`, {
                method: 'post',
                body: JSON.stringify(subscription),
            });

            return {data};
        } catch (error) {
            return {error};
        }
    };
};

export const editChannelSubscription = (subscription) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            const data = await doFetch(`${baseUrl}/api/v2/subscriptions/channel`, {
                method: 'put',
                body: JSON.stringify(subscription),
            });

            return {data};
        } catch (error) {
            return {error};
        }
    };
};

export const deleteChannelSubscription = (subscriptionId) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            const data = await doFetch(`${baseUrl}/api/v2/subscriptions/channel/${subscriptionId}`, {
                method: 'delete',
            });

            return {data};
        } catch (error) {
            return {error};
        }
    };
};

export const fetchChannelSubscriptions = (channelId) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        let data = null;
        try {
            data = await doFetch(`${baseUrl}/api/v2/subscriptions/channel/${channelId}`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS,
            channelId,
            data,
        });

        return {data};
    };
};
export function getSettings(getState) {
    let data;
    const baseUrl = getPluginServerRoute(getState());
    try {
        data = doFetch(`${baseUrl}/api/v2/settingsinfo`, {
            method: 'get',
        });
    } catch (error) {
        return {error};
    }

    return data;
}

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

        dispatch({
            type: ActionTypes.RECEIVED_INSTANCE_STATUS,
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

export const openChannelSettings = (channelId) => {
    return {
        type: ActionTypes.OPEN_CHANNEL_SETTINGS,
        data: {
            channelId,
        },
    };
};

export const closeChannelSettings = () => {
    return {
        type: ActionTypes.CLOSE_CHANNEL_SETTINGS,
    };
};

export function handleInstanceStatusChange(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_INSTANCE_STATUS,
            data: msg.data,
        });
    };
}

export function sendEphemeralPost(store, message, channelId) {
    const timestamp = Date.now();
    const post = {
        id: 'jiraPlugin' + Date.now(),
        user_id: store.getState().entities.users.currentUserId,
        channel_id: channelId || getCurrentChannelId(store.getState()),
        message,
        type: 'system_ephemeral',
        create_at: timestamp,
        update_at: timestamp,
        root_id: '',
        parent_id: '',
        props: {},
    };

    store.dispatch({
        type: PostTypes.RECEIVED_NEW_POST,
        data: post,
        channelId,
    });
}
