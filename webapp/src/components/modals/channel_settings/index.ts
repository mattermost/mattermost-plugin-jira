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
    fetchJiraIssueMetadataForProjects,
    clearIssueMetadata,
    sendEphemeralPost,
} from 'actions';

import {getChannelSubscriptions, getChannelIdWithSettingsOpen, getJiraProjectMetadata, getJiraIssueMetadata} from 'selectors';

import ChannelSettingsModal from './channel_settings';

const mapStateToProps = (state) => {
    const channelId = getChannelIdWithSettingsOpen(state);
    let channel = null;
    let omitDisplayName = false;

    if (channelId !== '') {
        channel = getChannel(state, channelId);
        omitDisplayName = isDirectChannel(channel) || isGroupChannel(channel);
    }

    const jiraIssueMetadata = getJiraIssueMetadata(state);
    const jiraProjectMetadata = getJiraProjectMetadata(state);

    const channelSubscriptions = getChannelSubscriptions(state)[channelId];

    return {
        omitDisplayName,
        channelSubscriptions,
        channel,
        jiraIssueMetadata,
        jiraProjectMetadata,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeChannelSettings,
    fetchJiraProjectMetadata,
    fetchJiraIssueMetadataForProjects,
    clearIssueMetadata,
    createChannelSubscription,
    fetchChannelSubscriptions,
    deleteChannelSubscription,
    editChannelSubscription,
    sendEphemeralPost,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(ChannelSettingsModal);
