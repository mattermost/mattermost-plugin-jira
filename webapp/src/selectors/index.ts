// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createSelector} from 'reselect';

import {getConfig} from 'mattermost-redux/selectors/entities/general';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/users';
import {GlobalState} from 'mattermost-redux/types/store';
import {UserProfile} from 'mattermost-redux/types/users';

import PluginId from 'plugin_id';
import {PluginState, CreateModal, IssueMetadata, ProjectMetadata, ChannelSubscriptions, PluginSettings, JiraInstanceType} from 'types/model';

const getPluginState = (state: GlobalState): PluginState => state['plugins-' + PluginId] || {};

export const getPluginServerRoute = (state: GlobalState): string => {
    const config = getConfig(state);

    let basePath = '/';
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
    (user: UserProfile) => {
        let locale = 'en';
        if (user && user.locale) {
            locale = user.locale;
        }

        return locale;
    }
);

export const isCreateModalVisible = (state: GlobalState): boolean => getPluginState(state).createModalVisible;

export const getCreateModal = (state: GlobalState): CreateModal => getPluginState(state).createModal;

export const isAttachCommentToIssueModalVisible = (state: GlobalState): boolean => getPluginState(state).attachCommentToIssueModalVisible;

export const getAttachCommentToIssueModalForPostId = (state: GlobalState): string => getPluginState(state).attachCommentToIssueModalForPostId;

export const getJiraIssueMetadata = (state: GlobalState): IssueMetadata => getPluginState(state).jiraIssueMetadata;

export const getJiraProjectMetadata = (state: GlobalState): ProjectMetadata => getPluginState(state).jiraProjectMetadata;

export const getChannelIdWithSettingsOpen = (state: GlobalState): string => getPluginState(state).channelIdWithSettingsOpen;

export const getChannelSubscriptions = (state: GlobalState): ChannelSubscriptions => getPluginState(state).channelSubscriptions;

export const isUserConnected = (state: GlobalState): boolean => getPluginState(state).userConnected;

export const isInstanceInstalled = (state: GlobalState): boolean => getPluginState(state).instanceInstalled;

export const getPluginSettings = (state: GlobalState): PluginSettings => getPluginState(state).pluginSettings;

export const getInstalledInstanceType = (state: GlobalState): JiraInstanceType => getPluginState(state).instanceType;
