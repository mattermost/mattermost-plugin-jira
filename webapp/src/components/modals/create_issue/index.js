// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {closeCreateModal, createIssue, fetchJiraIssueMetadata} from 'actions';
import {isCreateModalVisible, getCreateModal, getJiraIssueMetadata} from 'selectors';

import CreateIssue from './create_issue';

const mapStateToProps = (state) => {
    const {postId, description, channelId} = getCreateModal(state);
    const post = (postId) ? getPost(state, postId) : null;

    const jiraIssueMetadata = getJiraIssueMetadata(state);

    return {
        visible: isCreateModalVisible(state),
        jiraIssueMetadata,
        post,
        description,
        channelId,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeCreateModal,
    create: createIssue,
    fetchJiraIssueMetadata,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssue);
