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
    getDefaultUserInstanceID,
} from 'selectors';

import ChannelSettingsModal from './channel_settings';

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

    return {
        omitDisplayName,
        channelSubscriptions,
        channel,
        installedInstances,
        connectedInstances,
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

export default connect(mapStateToProps, mapDispatchToProps)(ChannelSettingsModal);
