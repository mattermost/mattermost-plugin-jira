// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {getCurrentTeam} from 'mattermost-redux/selectors/entities/teams';

import {closeCreateModal, createIssue, fetchJiraIssueMetadataForProjects, redirectConnect} from 'actions';
import {isCreateModalVisible, getCreateModal} from 'selectors';

import CreateIssue from './create_issue_modal';

const mapStateToProps = (state) => {
    const {postId, description, channelId} = getCreateModal(state);
    const post = (postId) ? getPost(state, postId) : null;
    const currentTeam = getCurrentTeam(state);

    return {
        visible: isCreateModalVisible(state),
        post,
        description,
        channelId,
        currentTeam,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeCreateModal,
    create: createIssue,
    fetchJiraIssueMetadataForProjects,
    redirectConnect,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssue);
