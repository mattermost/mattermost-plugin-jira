// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {closeAttachModal, attachIssue, fetchJiraIssueMetadata} from 'actions';
import {isAttachModalVisible, getAttachModalForPostId, getJiraIssueMetadata} from 'selectors';

import AttachIssue from './attach_issue';

const mapStateToProps = (state) => {
    const postId = getAttachModalForPostId(state);
    const post = getPost(state, postId);

    const jiraIssueMetadata = getJiraIssueMetadata(state);

    return {
        visible: isAttachModalVisible(state),
        jiraIssueMetadata,
        post,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeAttachModal,
    create: attachIssue,
    fetchJiraIssueMetadata,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachIssue);
