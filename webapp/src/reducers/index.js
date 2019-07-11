// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import ActionTypes from 'action_types';

function userConnected(state = false, action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.is_connected;
    default:
        return state;
    }
}

function instanceInstalled(state = false, action) {
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

const createModalVisible = (state = false, action) => {
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

const createModal = (state = '', action) => {
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

const attachCommentToIssueModalVisible = (state = false, action) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const attachCommentToIssueModalForPostId = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return action.data.postId;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return '';
    default:
        return state;
    }
};

const jiraIssueMetadata = (state = null, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_JIRA_ISSUE_METADATA:
        return action.data;
    case ActionTypes.CLEAR_JIRA_ISSUE_METADATA:
        return null;
    default:
        return state;
    }
};

const jiraProjectMetadata = (state = null, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_JIRA_PROJECT_METADATA:
        return action.data;
    default:
        return state;
    }
};

const channelIdWithSettingsOpen = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.OPEN_CHANNEL_SETTINGS:
        return action.data.channelId;
    case ActionTypes.CLOSE_CHANNEL_SETTINGS:
        return '';
    default:
        return state;
    }
};

const channelSubscripitons = (state = {}, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS: {
        const nextState = {...state};
        nextState[action.channelId] = action.data;
        return nextState;
    }
    default:
        return state;
    }
};

export default combineReducers({
    userConnected,
    instanceInstalled,
    createModalVisible,
    createModal,
    attachCommentToIssueModalVisible,
    attachCommentToIssueModalForPostId,
    jiraIssueMetadata,
    jiraProjectMetadata,
    channelIdWithSettingsOpen,
    channelSubscripitons,
});
