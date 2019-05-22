// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getChannel} from 'mattermost-redux/selectors/entities/channels';

import {createChannelSubscription, fetchChannelSubscriptions, deleteChannelSubscription, editChannelSubscription, closeChannelSettings, fetchJiraIssueMetadata} from 'actions';
import {getChannelSubscriptions, getChannelIdWithSettingsOpen, getJiraIssueMetadata} from 'selectors';

import ChannelSettingsModal from './channel_settings';

const mapStateToProps = (state) => {
    const channelId = getChannelIdWithSettingsOpen(state);
    let channel = null;

    if (channelId !== '') {
        channel = getChannel(state, channelId);
    }

    const jiraMetadata = getJiraIssueMetadata(state);

    const channelSubscriptions = getChannelSubscriptions(state)[channelId];

    return {
        channelSubscriptions,
        channel,
        jiraMetadata,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeChannelSettings,
    getMetadata: fetchJiraIssueMetadata,
    createChannelSubscription,
    fetchChannelSubscriptions,
    deleteChannelSubscription,
    editChannelSubscription,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(ChannelSettingsModal);
