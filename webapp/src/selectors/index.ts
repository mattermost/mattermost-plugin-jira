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

export const getJiraIssueMetadata = (state) => getPluginState(state).jiraIssueMetadata;

export const getJiraProjectMetadata = (state) => getPluginState(state).jiraProjectMetadata;

export const getChannelIdWithSettingsOpen = (state) => getPluginState(state).channelIdWithSettingsOpen;

export const getChannelSubscriptions = (state) => getPluginState(state).channelSubscriptions;

export const isUserConnected = (state) => getPluginState(state).userConnected;

export const canUserConnect = (state) => getPluginState(state).userCanConnect;

export const getUserConnectedInstances = (state): Instance[] => getPluginState(state).userConnectedInstances;

export const getInstalledInstances = (state): Instance[] => getPluginState(state).installedInstances;

export const getDefaultConnectInstance = (state) => getPluginState(state).defaultConnectInstance;

export const getDefaultUserInstance = (state) => getPluginState(state).defaultUserInstance;

export const getPluginSettings = (state) => getPluginState(state).pluginSettings;
