// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import {CreateTypes} from 'action_types';

function connected(state = false, action) {
    switch (action.type) {
    case CreateTypes.RECEIVED_CONNECTED:
        return action.data.connected;
    default:
        return state;
    }
}

const createModalVisible = (state = false, action) => {
    switch (action.type) {
    case CreateTypes.OPEN_CREATE_MODAL:
        return true;
    case CreateTypes.CLOSE_CREATE_MODAL:
        return false;
    default:
        return state;
    }
};

const createModalForPostId = (state = '', action) => {
    switch (action.type) {
    case CreateTypes.OPEN_CREATE_MODAL:
        return action.data.postId;
    case CreateTypes.CLOSE_CREATE_MODAL:
        return '';
    default:
        return state;
    }
};

export default combineReducers({
    connected,
    createModalVisible,
    createModalForPostId,
});