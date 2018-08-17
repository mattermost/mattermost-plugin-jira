// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import {CreateTypes} from 'action_types';

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
    createModalVisible,
    createModalForPostId
});