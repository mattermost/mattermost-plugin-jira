// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import ActionTypes from 'action_types';

function connected(state = false, action) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.is_connected;
    default:
        return state;
    }
}

const createModalVisible = (state = false, action) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_MODAL:
        return true;
    case ActionTypes.CLOSE_CREATE_MODAL:
        return false;
    default:
        return state;
    }
};

const createModalForPostId = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_MODAL:
        return action.data.postId;
    case ActionTypes.CLOSE_CREATE_MODAL:
        return '';
    default:
        return state;
    }
};

const jiraMetadata = (state = null, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_METADATA:
        return action.data;
    default:
        return state;
    }
};

export default combineReducers({
    connected,
    createModalVisible,
    createModalForPostId,
    jiraMetadata,
});
