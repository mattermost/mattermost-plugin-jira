// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getChannel} from 'mattermost-redux/selectors/entities/channels';
import {isDirectChannel, isGroupChannel} from 'mattermost-redux/utils/channel_utils';

import {
    createChannelSubscription,
    fetchChannelSubscriptions,
    deleteChannelSubscription,
    editChannelSubscription,
    closeChannelSettings,
    fetchJiraProjectMetadata,
    fetchJiraProjectMetadataForAllInstances,
    fetchJiraIssueMetadataForProjects,
    sendEphemeralPost,
    getConnected,
} from 'actions';

import {
    getChannelSubscriptions,
    getChannelIdWithSettingsOpen,
    getInstalledInstances,
    getUserConnectedInstances,
    getPluginSettings,
} from 'selectors';

import ChannelSubscriptionsModal from './channel_subscriptions';

const mapStateToProps = (state) => {
    const channelId = getChannelIdWithSettingsOpen(state);
    let channel = null;
    let omitDisplayName = false;

    if (channelId !== '') {
        channel = getChannel(state, channelId);
        omitDisplayName = isDirectChannel(channel) || isGroupChannel(channel);
    }

    const channelSubscriptions = getChannelSubscriptions(state)[channelId];

    const installedInstances = getInstalledInstances(state);
    const connectedInstances = getUserConnectedInstances(state);
    const pluginSettings = getPluginSettings(state);
    const securityLevelEmptyForJiraSubscriptions = pluginSettings.security_level_empty_for_jira_subscriptions;

    return {
        omitDisplayName,
        channelSubscriptions,
        channel,
        installedInstances,
        connectedInstances,
        securityLevelEmptyForJiraSubscriptions,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeChannelSettings,
    fetchJiraProjectMetadata,
    fetchJiraProjectMetadataForAllInstances,
    fetchJiraIssueMetadataForProjects,
    createChannelSubscription,
    fetchChannelSubscriptions,
    deleteChannelSubscription,
    editChannelSubscription,
    getConnected,
    sendEphemeralPost,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(ChannelSubscriptionsModal);
