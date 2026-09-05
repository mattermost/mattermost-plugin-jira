// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers} from 'redux';

import ActionTypes from '../action_types';
import {ChannelSubscription, PluginSettings} from 'types/model';
import {
    RHSErrorCode,
    RHSIssue,
    RHSSort,
    RHSTab,
    RHS_DEFAULT_SORT,
    RHS_DEFAULT_TAB,
} from 'types/rhs';

export type Action<T extends string = string> = {
    type: T
}

export interface AnyAction extends Action {
    [key: string]: any
}

export interface AnyState {
    [key: string]: any
}

function installedInstances(state = [], action = {} as AnyAction) {
    // We're notified of the instance status at startup (through getConnected)
    // and when we get a websocket instance_status event
    switch (action.type) {
    case ActionTypes.RECEIVED_INSTANCE_STATUS:
        return action.data.instances ? action.data.instances : [];
    default:
        return state;
    }
}

function userConnected(state = false, action = {} as AnyAction) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.is_connected;
    default:
        return state;
    }
}

function userCanConnect(state = false, action = {} as AnyAction) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.can_connect;
    default:
        return state;
    }
}

function defaultUserInstanceID(state = '', action = {} as AnyAction) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data.user_info && action.data.user_info.default_instance_id) {
            return action.data.user_info.default_instance_id;
        }
        return state;
    default:
        return state;
    }
}

function userConnectedInstances(state = [], action = {} as AnyAction) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data.user_info) {
            return action.data.user_info.connected_instances ? action.data.user_info.connected_instances : [];
        }
        return state;
    default:
        return state;
    }
}

function pluginSettings(state: PluginSettings | null = null, action = {} as AnyAction): PluginSettings | null {
    switch (action.type) {
    case ActionTypes.RECEIVED_PLUGIN_SETTINGS:
        return action.data;
    default:
        return state;
    }
}

const connectModalVisible = (state = false, action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.OPEN_CONNECT_MODAL:
        return true;
    case ActionTypes.CLOSE_CONNECT_MODAL:
        return false;
    default:
        return state;
    }
};

const disconnectModalVisible = (state = false, action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.OPEN_DISCONNECT_MODAL:
        return true;
    case ActionTypes.CLOSE_DISCONNECT_MODAL:
        return false;
    default:
        return state;
    }
};

const createModalVisible = (state = false, action = {} as AnyAction) => {
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

const createModal = (state = '', action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST:
        return {
            state,
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

const attachCommentToIssueModalVisible = (state = false, action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const attachCommentToIssueModalForPostId = (state = '', action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return action.data.postId;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return '';
    default:
        return state;
    }
};

const channelIdWithSettingsOpen = (state = '', action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.OPEN_CHANNEL_SETTINGS:
        return action.data.channelId;
    case ActionTypes.CLOSE_CHANNEL_SETTINGS:
        return '';
    default:
        return state;
    }
};

const subscriptionTemplates = (state = {} as AnyState, action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_SUBSCRIPTION_TEMPLATES: {
        const nextState = {...state};
        nextState.subscriptionTemplates = action.data;
        return nextState;
    }
    case ActionTypes.DELETED_SUBSCRIPTION_TEMPLATE: {
        const subTemplate = action.data;
        const nextState = {...state};

        nextState.subscriptionTemplates = nextState.subscriptionTemplates.filter((st: { id: string; }) => {
            return st.id !== subTemplate.id;
        });

        return nextState;
    }
    case ActionTypes.CREATED_SUBSCRIPTION_TEMPLATE: {
        const subTemplate = action.data;
        const nextState = {...state};
        nextState.subscriptionTemplates = [...nextState.subscriptionTemplates, subTemplate];

        return nextState;
    }
    case ActionTypes.EDITED_SUBSCRIPTION_TEMPLATE: {
        const subTemplate = action.data;
        const nextState = {...state};

        const index = nextState.subscriptionTemplates.findIndex((template: { id: string; }) => template.id === subTemplate.id);

        const newArray = [...nextState.subscriptionTemplates];
        newArray[index] = subTemplate;

        return {
            ...nextState,
            subscriptionTemplates: newArray,
        };
    }
    default:
        return state;
    }
};

const subscriptionTemplatesForProjectKey = (state = {} as AnyState, action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_SUBSCRIPTION_TEMPLATES_PROJECT_KEY: {
        const nextState = {...state};
        nextState.subscriptionTemplatesForProjectKey = action.data;
        return nextState;
    }
    default:
        return state;
    }
};

const channelSubscriptions = (state = {} as AnyState, action = {} as AnyAction) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS: {
        const nextState = {...state};
        nextState[action.channelId] = action.data;
        return nextState;
    }
    case ActionTypes.DELETED_CHANNEL_SUBSCRIPTION: {
        const sub = action.data;
        const newSubs = state[sub.channel_id].concat([]);

        if (!newSubs) {
            return {...state};
        }

        newSubs.splice(newSubs.findIndex((s: ChannelSubscription) => s.id === sub.id), 1);

        return {
            ...state,
            [sub.channel_id]: newSubs,
        };
    }
    case ActionTypes.CREATED_CHANNEL_SUBSCRIPTION: {
        const sub = action.data;
        const newSubs = state[sub.channel_id] ? state[sub.channel_id].concat([]) : [];
        newSubs.push(sub);

        return {
            ...state,
            [sub.channel_id]: newSubs,
        };
    }
    case ActionTypes.EDITED_CHANNEL_SUBSCRIPTION: {
        const sub = action.data;
        const newSubs = state[sub.channel_id] ? state[sub.channel_id].concat([]) : [];
        newSubs.splice(newSubs.findIndex((s: ChannelSubscription) => s.id === sub.id), 1, sub);

        return {
            ...state,
            [sub.channel_id]: newSubs,
        };
    }
    default:
        return state;
    }
};

export type RHSState = {
    instanceID: string;
    tab: RHSTab;
    sort: RHSSort;
    issues: RHSIssue[];
    tabs: RHSTab[];
    nextPageToken: string;
    isLast: boolean;
    loading: boolean;
    error: RHSErrorCode | null;
    requestId: number;
};

const defaultRHSState: RHSState = {
    instanceID: '',
    tab: RHS_DEFAULT_TAB,
    sort: RHS_DEFAULT_SORT,
    issues: [],
    tabs: [],
    nextPageToken: '',
    isLast: true,
    loading: false,
    error: null,
    requestId: 0,
};

function rhs(state: RHSState = defaultRHSState, action = {} as AnyAction): RHSState {
    switch (action.type) {
    case ActionTypes.SET_RHS_VIEW:
        return {
            ...state,
            instanceID: action.data.instanceID,
            tab: action.data.tab,
            sort: action.data.sort,
        };
    case ActionTypes.RHS_ISSUES_LOADING: {
        const reset = Boolean(action.data && action.data.reset);
        return {
            ...state,
            requestId: state.requestId + 1,
            loading: true,
            error: null,
            ...(reset ? {
                issues: [],
                nextPageToken: '',
                isLast: true,
            } : {}),
        };
    }
    case ActionTypes.RECEIVED_RHS_ISSUES:
        if (action.requestId !== state.requestId) {
            return state;
        }
        return {
            ...state,
            issues: action.data.issues ? action.data.issues : [],
            tabs: action.data.tabs ? action.data.tabs : [],
            nextPageToken: action.data.nextPageToken ? action.data.nextPageToken : '',
            isLast: Boolean(action.data.isLast),
            loading: false,
            error: null,
        };
    case ActionTypes.RECEIVED_RHS_ISSUES_APPEND:
        if (action.requestId !== state.requestId) {
            return state;
        }
        return {
            ...state,
            issues: state.issues.concat(action.data.issues ? action.data.issues : []),
            tabs: action.data.tabs ? action.data.tabs : state.tabs,
            nextPageToken: action.data.nextPageToken ? action.data.nextPageToken : '',
            isLast: Boolean(action.data.isLast),
            loading: false,
            error: null,
        };
    case ActionTypes.RHS_ISSUES_ERROR:
        if (action.requestId !== state.requestId) {
            return state;
        }
        return {
            ...state,
            loading: false,
            error: action.data,
        };
    default:
        return state;
    }
}

export default combineReducers({
    userConnected,
    userCanConnect,
    userConnectedInstances,
    installedInstances,
    defaultUserInstanceID,
    pluginSettings,
    connectModalVisible,
    disconnectModalVisible,
    createModalVisible,
    createModal,
    attachCommentToIssueModalVisible,
    attachCommentToIssueModalForPostId,
    channelIdWithSettingsOpen,
    subscriptionTemplates,
    subscriptionTemplatesForProjectKey,
    channelSubscriptions,
    rhs,
});
