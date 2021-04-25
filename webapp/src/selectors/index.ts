// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createSelector} from 'reselect';

import {getConfig} from 'mattermost-redux/selectors/entities/general';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/users';

import PluginId from 'plugin_id';
import {Instance} from 'types/model';

const getPluginState = (state) => state['plugins-' + PluginId] || {};

export const getPluginServerRoute = (state) => {
    const config = getConfig(state);

    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;

        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath + '/plugins/' + PluginId;
};

export const getCurrentUserLocale = createSelector(
    getCurrentUser,
    (user) => {
        let locale = 'en';
        if (user && user.locale) {
            locale = user.locale;
        }

        return locale;
    }
);

export const isConnectModalVisible = (state) => getPluginState(state).connectModalVisible;
export const isDisconnectModalVisible = (state) => getPluginState(state).disconnectModalVisible;

export const isCreateModalVisible = (state) => getPluginState(state).createModalVisible;

export const getCreateModal = (state) => getPluginState(state).createModal;

export const isAttachCommentToIssueModalVisible = (state) => getPluginState(state).attachCommentToIssueModalVisible;

export const getAttachCommentToIssueModalForPostId = (state) => getPluginState(state).attachCommentToIssueModalForPostId;

export const getChannelIdWithSettingsOpen = (state) => getPluginState(state).channelIdWithSettingsOpen;

export const getChannelSubscriptions = (state) => getPluginState(state).channelSubscriptions;

export const isUserConnected = (state) => getUserConnectedInstances(state).length > 0;

export const canUserConnect = (state) => getPluginState(state).userCanConnect;

export const getUserConnectedInstances = (state): Instance[] => {
    const installed = getPluginState(state).installedInstances as Instance[];
    const connected = getPluginState(state).userConnectedInstances as Instance[];
    if (!installed || !connected) {
        return [];
    }

    return connected.filter((instance1) => installed.find((instance2) => instance1.instance_id === instance2.instance_id));
};

export const getInstalledInstances = (state): Instance[] => getPluginState(state).installedInstances;
export const instanceIsInstalled = (state): boolean => getInstalledInstances(state).length > 0;

export const getDefaultUserInstanceID = (state) => getPluginState(state).defaultUserInstanceID;

export const getPluginSettings = (state) => getPluginState(state).pluginSettings;
