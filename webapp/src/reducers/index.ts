// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import {
    GenericAction,
} from 'mattermost-redux/types/actions';

import ActionTypes from 'action_types';

function userConnected(state = false, action: GenericAction): boolean {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.is_connected;
    default:
        return state;
    }
}

function instanceInstalled(state = false, action: GenericAction): boolean {
    // We're notified of the instance status at startup (through getConnected)
    // and when we get a websocket instance_status event
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.instance_installed ? action.data.instance_installed : state;
    case ActionTypes.RECEIVED_INSTANCE_STATUS:
        return action.data.instance_installed;
    default:
        return state;
    }
}

function instanceType(state = '', action: GenericAction): string {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.instance_type ? action.data.instance_type : state;
    case ActionTypes.RECEIVED_INSTANCE_STATUS:
        return action.data.instance_type;
    default:
        return state;
    }
}

function pluginSettings(state = null, action: GenericAction): any {
    switch (action.type) {
    case ActionTypes.RECEIVED_PLUGIN_SETTINGS:
        return action.data;
    default:
        return state;
    }
}

const createModalVisible = (state = false, action: GenericAction): boolean => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST:
        return true;
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

// TODO: Crosscheck types
const createModal = (state = '', action: GenericAction): any => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST:
        return {
            ...state,
            postId: action.data.postId,
            description: action.data.description,
            channelId: action.data.channelId,
        };
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return {};
    default:
        return state;
    }
};

const attachCommentToIssueModalVisible = (state = false, action: GenericAction): boolean => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const attachCommentToIssueModalForPostId = (state = '', action: GenericAction): string => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return action.data.postId;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return '';
    default:
        return state;
    }
};

const jiraIssueMetadata = (state = null, action: GenericAction): any => {
    switch (action.type) {
    case ActionTypes.RECEIVED_JIRA_ISSUE_METADATA:
        return action.data;
    case ActionTypes.CLEAR_JIRA_ISSUE_METADATA:
        return null;
    case ActionTypes.CLOSE_CHANNEL_SETTINGS:
        return null;
    default:
        return state;
    }
};

const jiraProjectMetadata = (state = null, action: GenericAction): any => {
    switch (action.type) {
    case ActionTypes.RECEIVED_JIRA_PROJECT_METADATA:
        return action.data;
    case ActionTypes.CLOSE_CHANNEL_SETTINGS:
        return null;
    default:
        return state;
    }
};

const channelIdWithSettingsOpen = (state = '', action: GenericAction): string => {
    switch (action.type) {
    case ActionTypes.OPEN_CHANNEL_SETTINGS:
        return action.data.channelId;
    case ActionTypes.CLOSE_CHANNEL_SETTINGS:
        return '';
    default:
        return state;
    }
};

const channelSubscriptions = (state = {}, action: GenericAction): any => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS: {
        const nextState = {...state};
        nextState[action.channelId] = action.data;
        return nextState;
    }
    case ActionTypes.DELETED_CHANNEL_SUBSCRIPTION: {
        const sub = action.data;
        const newSubs = state[sub.channel_id].concat([]);
        newSubs.splice(newSubs.findIndex((s) => s.id === sub.id), 1);

        return {
            ...state,
            [sub.channel_id]: newSubs,
        };
    }
    case ActionTypes.CREATED_CHANNEL_SUBSCRIPTION: {
        const sub = action.data;
        const newSubs = state[sub.channel_id].concat([]);
        newSubs.push(sub);

        return {
            ...state,
            [sub.channel_id]: newSubs,
        };
    }
    case ActionTypes.EDITED_CHANNEL_SUBSCRIPTION: {
        const sub = action.data;
        const newSubs = state[sub.channel_id].concat([]);
        newSubs.splice(newSubs.findIndex((s) => s.id === sub.id), 1, sub);

        return {
            ...state,
            [sub.channel_id]: newSubs,
        };
    }
    default:
        return state;
    }
};

export default combineReducers({
    userConnected,
    instanceInstalled,
    instanceType,
    pluginSettings,
    createModalVisible,
    createModal,
    attachCommentToIssueModalVisible,
    attachCommentToIssueModalForPostId,
    jiraIssueMetadata,
    jiraProjectMetadata,
    channelIdWithSettingsOpen,
    channelSubscriptions,
});
