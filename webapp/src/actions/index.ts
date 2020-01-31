// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PostTypes} from 'mattermost-redux/action_types';
import {
    GenericAction,
    DispatchFunc,
    ActionFunc,
    GetStateFunc,
    ActionResult,
} from 'mattermost-redux/types/actions';
import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import ActionTypes from 'action_types';
import {doFetch, doFetchWithResponse, buildQueryString} from 'client';
import {getPluginServerRoute} from 'selectors';

export const openCreateModal = (postId: string): GenericAction => {
    return {
        type: ActionTypes.OPEN_CREATE_ISSUE_MODAL,
        data: {
            postId,
        },
    };
};

// TODO: Crosscheck return type
export const openCreateModalWithoutPost = (description: string, channelId: string) => (dispatch: DispatchFunc): DispatchFunc => dispatch({
    type: ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST,
    data: {
        description,
        channelId,
    },
});

export const closeCreateModal = (): GenericAction => {
    return {
        type: ActionTypes.CLOSE_CREATE_ISSUE_MODAL,
    };
};

export const openAttachCommentToIssueModal = (postId: string): GenericAction => {
    return {
        type: ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL,
        data: {
            postId,
        },
    };
};

export const closeAttachCommentToIssueModal = (): GenericAction => {
    return {
        type: ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL,
    };
};

export const fetchJiraIssueMetadataForProjects = (projectKeys: Array<string>): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export const clearIssueMetadata = (): ActionFunc => {
    return async (dispatch: DispatchFunc): Promise<ActionResult|ActionResult[]> => {
        dispatch({type: ActionTypes.CLEAR_JIRA_ISSUE_METADATA});
    };
};

export const fetchJiraProjectMetadata = (): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

// TODO: Explore what type params are
export const searchIssues = (params: any): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): Promise<ActionResult|ActionResult[]> => {
        const url = getPluginServerRoute(getState()) + '/api/v2/get-search-issues';
        return doFetchWithResponse(`${url}${buildQueryString(params)}`);
    };
};

export const createIssue = (payload: any): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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
export const attachCommentToIssue = (payload: any): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export const createChannelSubscription = (subscription: any): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export const editChannelSubscription = (subscription: any): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export const deleteChannelSubscription = (subscription: any): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export const fetchChannelSubscriptions = (channelId: string): ActionFunc => {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export function getSettings(): ActionFunc {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export function getConnected(): ActionFunc {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): ActionResult => {
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

export const openChannelSettings = (channelId: string): GenericAction => {
    return {
        type: ActionTypes.OPEN_CHANNEL_SETTINGS,
        data: {
            channelId,
        },
    };
};

export const closeChannelSettings = (): GenericAction => {
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

export function sendEphemeralPost(message: string, channelId: string) {
    return (dispatch: DispatchFunc, getState: GetStateFunc): void => {
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
