// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createSelector} from 'reselect';

import {getConfig} from 'mattermost-redux/selectors/entities/general';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/users';

import manifest from 'manifest';

import {Instance} from 'types/model';
import {GlobalState, pluginStateKey} from 'types/store';

const getPluginState = (state: GlobalState) => state[pluginStateKey] || {};

export const getPluginServerRoute = (state: GlobalState) => {
    const config = getConfig(state);
    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;

        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath + '/plugins/' + manifest.id;
};

export const getCurrentUserLocale = createSelector(
    getCurrentUser,
    (user) => {
        let locale = 'en';
        if (user && user.locale) {
            locale = user.locale;
        }

        return locale;
    },
);

export const isConnectModalVisible = (state: GlobalState) => getPluginState(state).connectModalVisible;
export const isDisconnectModalVisible = (state: GlobalState) => getPluginState(state).disconnectModalVisible;

export const isCreateModalVisible = (state: GlobalState) => getPluginState(state).createModalVisible;

export const getCreateModal = (state: GlobalState) => getPluginState(state).createModal;

export const isAttachCommentToIssueModalVisible = (state: GlobalState) => getPluginState(state).attachCommentToIssueModalVisible;

export const getAttachCommentToIssueModalForPostId = (state: GlobalState) => getPluginState(state).attachCommentToIssueModalForPostId;

export const getChannelIdWithSettingsOpen = (state: GlobalState) => getPluginState(state).channelIdWithSettingsOpen;

export const getChannelSubscriptions = (state: GlobalState) => getPluginState(state).channelSubscriptions;

export const isUserConnected = (state: GlobalState) => getUserConnectedInstances(state).length > 0;

export const canUserConnect = (state: GlobalState) => getPluginState(state).userCanConnect;

export const getUserConnectedInstances = (state: GlobalState): Instance[] => {
    const installed = getPluginState(state).installedInstances as Instance[];
    const connected = getPluginState(state).userConnectedInstances as Instance[];
    if (!installed || !connected) {
        return [];
    }

    return connected.filter((instance1) => installed.find((instance2) => instance1.instance_id === instance2.instance_id));
};

export const getInstalledInstances = (state: GlobalState): Instance[] => getPluginState(state).installedInstances;
export const instanceIsInstalled = (state: GlobalState): boolean => getInstalledInstances(state).length > 0;

export const getDefaultUserInstanceID = (state: GlobalState) => getPluginState(state).defaultUserInstanceID;

export const getPluginSettings = (state: GlobalState) => getPluginState(state).pluginSettings;
