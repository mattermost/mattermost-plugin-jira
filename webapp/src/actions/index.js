// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PostTypes} from 'mattermost-redux/action_types';
import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import ActionTypes from 'action_types';
import {doFetch, doFetchWithResponse, buildQueryString} from 'client';
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

/**
 * Returns list of statuses the jira project uses, list is not stored in store but returned from function
 * @function fetchJiraProjectStatuses
 * @param none
 * @returns {Promise} Promise object represents list data or error
 */
export const fetchJiraProjectStatuses = () => {
    return async (dispatch, getState) => {
        const baseURL = getPluginServerRoute(getState());
        let data = null;
        try {
            data = await doFetch(`${baseURL}/api/v2/get-all-statuses`, {
                message: 'get',
            });
        } catch (error) {
            return {error};
        }

        return {data};
    };
};

export const fetchJiraIssueMetadataForProjects = (projectKeys) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        const projectKeysParam = projectKeys.join(',');
        let data = null;
        try {
            data = await doFetch(`${baseUrl}/api/v2/get-create-issue-metadata-for-project?project-keys=${projectKeysParam}`, {
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

export const clearIssueMetadata = () => {
    return async (dispatch) => {
        dispatch({type: ActionTypes.CLEAR_JIRA_ISSUE_METADATA});
    };
};

export const fetchJiraProjectMetadata = () => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        let data = null;
        try {
            data = await doFetch(`${baseUrl}/api/v2/get-jira-project-metadata`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_JIRA_PROJECT_METADATA,
            data,
        });

        return {data};
    };
};

export const searchIssues = (params) => {
    return async (dispatch, getState) => {
        const url = getPluginServerRoute(getState()) + '/api/v2/get-search-issues';
        return doFetchWithResponse(`${url}${buildQueryString(params)}`);
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

            dispatch({
                type: ActionTypes.CREATED_CHANNEL_SUBSCRIPTION,
                data,
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

            dispatch({
                type: ActionTypes.EDITED_CHANNEL_SUBSCRIPTION,
                data,
            });

            return {data};
        } catch (error) {
            return {error};
        }
    };
};

export const deleteChannelSubscription = (subscription) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            await doFetch(`${baseUrl}/api/v2/subscriptions/channel/${subscription.id}`, {
                method: 'delete',
            });

            dispatch({
                type: ActionTypes.DELETED_CHANNEL_SUBSCRIPTION,
                data: subscription,
            });

            return {data: subscription};
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

export function getSettings() {
    return async (dispatch, getState) => {
        let data;
        const baseUrl = getPluginServerRoute(getState());
        try {
            data = await doFetch(`${baseUrl}/api/v2/settingsinfo`, {
                method: 'get',
            });

            dispatch({
                type: ActionTypes.RECEIVED_PLUGIN_SETTINGS,
                data,
            });
        } catch (error) {
            return {error};
        }

        return data;
    };
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
        // Update the user's UI state when the instance state changes
        getConnected()(store.dispatch, store.getState);

        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_INSTANCE_STATUS,
            data: msg.data,
        });
    };
}

export function sendEphemeralPost(message, channelId) {
    return (dispatch, getState) => {
        const timestamp = Date.now();
        const post = {
            id: 'jiraPlugin' + Date.now(),
            user_id: getState().entities.users.currentUserId,
            channel_id: channelId || getCurrentChannelId(getState()),
            message,
            type: 'system_ephemeral',
            create_at: timestamp,
            update_at: timestamp,
            root_id: '',
            parent_id: '',
            props: {},
        };

        dispatch({
            type: PostTypes.RECEIVED_NEW_POST,
            data: post,
            channelId,
        });
    };
}
