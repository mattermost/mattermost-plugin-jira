// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PostTypes} from 'mattermost-redux/action_types';
import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import PluginId from 'plugin_id';
import ActionTypes from 'action_types';
import {doFetch, doFetchWithResponse, buildQueryString} from 'client';
import {getPluginServerRoute, getInstalledInstances, getUserConnectedInstances} from 'selectors';
import {isDesktopApp, isMinimumDesktopAppVersion} from 'utils/user_agent';
import {ChannelSubscription, CreateIssueRequest, SearchIssueParams, InstanceType} from 'types/model';

export const openConnectModal = () => {
    return {
        type: ActionTypes.OPEN_CONNECT_MODAL,
    };
};

export const closeConnectModal = () => {
    return {
        type: ActionTypes.CLOSE_CONNECT_MODAL,
    };
};

export const openDisconnectModal = () => {
    return {
        type: ActionTypes.OPEN_DISCONNECT_MODAL,
    };
};

export const closeDisconnectModal = () => {
    return {
        type: ActionTypes.CLOSE_DISCONNECT_MODAL,
    };
};

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

export const fetchJiraIssueMetadataForProjects = (projectKeys: string[], instanceID: string) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        const projectKeysParam = projectKeys.join(',');
        let data = null;
        const params = `project-keys=${projectKeysParam}&instance_id=${instanceID}`;
        try {
            data = await doFetch(`${baseUrl}/api/v2/get-create-issue-metadata-for-project?${params}`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }

        if (data.error) {
            return {error: new Error(data.error)};
        }

        return {data};
    };
};

export const fetchJiraProjectMetadata = (instanceID: string) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        let data = null;
        try {
            data = await doFetch(`${baseUrl}/api/v2/get-jira-project-metadata?instance_id=${instanceID}`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }

        if (data.error) {
            return {error: new Error(data.error)};
        }

        return {data};
    };
};

export const searchIssues = (params: SearchIssueParams) => {
    return async (dispatch, getState) => {
        const url = getPluginServerRoute(getState()) + '/api/v2/get-search-issues';
        return doFetchWithResponse(`${url}${buildQueryString(params)}`);
    };
};

export const createIssue = (payload: CreateIssueRequest) => {
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

export const createChannelSubscription = (subscription: ChannelSubscription) => {
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

export const editChannelSubscription = (subscription: ChannelSubscription) => {
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

export const deleteChannelSubscription = (subscription: ChannelSubscription) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        try {
            await doFetch(`${baseUrl}/api/v2/subscriptions/channel/${subscription.id}?instance_id=${subscription.instance_id}`, {
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

export const fetchChannelSubscriptions = (channelId: string) => {
    return async (dispatch, getState) => {
        const baseUrl = getPluginServerRoute(getState());
        const connectedInstances = getUserConnectedInstances(getState());

        const promises = connectedInstances.map((instance) => {
            return doFetch(`${baseUrl}/api/v2/subscriptions/channel/${channelId}?instance_id=${instance.instance_id}`, {
                method: 'get',
            });
        });

        let allResponses;
        try {
            allResponses = await Promise.allSettled(promises);
        } catch (error) {
            return {error};
        }

        const errors: string[] = [];
        let data: ChannelSubscription[] = [];

        for (const res of allResponses) {
            if (res.status === 'rejected') {
                errors.push(res.reason);
            } else {
                data = data.concat(res.value);
            }
        }

        if (errors.length > 0 && allResponses.length === errors.length) {
            return {error: new Error(errors[0])};
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

export function disconnectUser(instanceID: string) {
    return async (dispatch, getState) => {
        let data;
        const baseUrl = getPluginServerRoute(getState());
        try {
            data = await doFetch(`${baseUrl}/api/v3/disconnect`, {
                method: 'post',
                body: JSON.stringify({instance_id: instanceID}),
            });
        } catch (error) {
            return {error};
        }

        return dispatch(getConnected());
    };
}

export function handleConnectFlow(instanceID?: string) {
    return async (dispatch, getState) => {
        const state = getState();
        const instances = getInstalledInstances(state);
        const connectedInstances = getUserConnectedInstances(state);

        if (!instances.length) {
            dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
            return;
        }

        // TODO <> connectedInstances is still 2 after uninstalling an instance
        if (instances.length === connectedInstances.length) {
            let postMessage = `Your Mattermost account is already linked to ${instances[0].instance_id}\n`;
            if (instances.length > 1) {
                const bullets = connectedInstances.map((instance) => `* ${instance.instance_id}`).join('\n');
                postMessage = `Your Mattermost account is already linked to all installed Jira instances:\n${bullets}\n`;
            }
            postMessage += 'Please use `/jira disconnect` to disconnect.';
            dispatch(sendEphemeralPost(postMessage));
            return;
        }

        let instance;
        if (instances.length === 1) {
            instance = instances[0];
        }
        if (instanceID) {
            const alreadyConnected = connectedInstances.find((i) => i.instance_id === instanceID);

            if (alreadyConnected) {
                dispatch(sendEphemeralPost(
                    'Your Jira account at ' + instanceID + ' is already linked to your Mattermost account. Please use `/jira disconnect` to disconnect.'));
                return;
            }

            instance = instances.find((i) => i.instance_id === instanceID);
            if (!instance) {
                const errMsg = 'Jira instance ' + instanceID + ' is not installed. Please type `/jira instance list` to see the available Jira instances.';
                dispatch(sendEphemeralPost(errMsg));
                return;
            }
        }

        if (instance && instance.type === InstanceType.SERVER && isDesktopApp() && !isMinimumDesktopAppVersion(4, 3, 0)) { // eslint-disable-line no-magic-numbers
            const errMsg = 'Your version of the Mattermost desktop client does not support authenticating between Jira and Mattermost directly. To connect your Jira account with Mattermost, please go to Mattermost via your web browser and type `/jira connect`, or [check the Mattermost download page](https://mattermost.com/download/#mattermostApps) to get the latest version of the desktop client.';
            dispatch(sendEphemeralPost(errMsg));
            return;
        }

        if (instance && instance.instance_id) {
            dispatch(redirectConnect(instance.instance_id));
            return;
        }

        dispatch(openConnectModal());
    };
}

export function redirectConnect(instanceID: string) {
    return async (dispatch, getState) => {
        const instancePrefix = '/instance/' + btoa(instanceID);
        const target = '/plugins/' + PluginId + instancePrefix + '/user/connect';
        window.open(target, '_blank');
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

export function sendEphemeralPost(message: string, channelId?: string) {
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
