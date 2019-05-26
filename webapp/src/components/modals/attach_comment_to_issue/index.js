// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {closeAttachCommentToIssueModal, attachCommentToIssue, fetchJiraIssueMetadata, fetchJiraIssues} from 'actions';
import {
    isAttachCommentToIssueModalVisible,
    getAttachCommentToIssueModalForPostId,
    getJiraIssueMetadata,
    getJiraIssueOptions,
} from 'selectors';

import AttachCommentToIssue from './attach_comment_to_issue';

const mapStateToProps = (state) => {
    const postId = getAttachCommentToIssueModalForPostId(state);
    const post = getPost(state, postId);

    const jiraIssueMetadata = getJiraIssueMetadata(state);
    const jiraIssueOptions = getJiraIssueOptions(state);

    return {
        visible: isAttachCommentToIssueModalVisible(state),
        jiraIssueMetadata,
        jiraIssueOptions,
        post,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeAttachCommentToIssueModal,
    create: attachCommentToIssue,
    fetchJiraIssueMetadata,
    fetchJiraIssues,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssue);
