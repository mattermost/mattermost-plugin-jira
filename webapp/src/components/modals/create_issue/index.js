// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {closeCreateModal, createIssue, getCreateIssueMetadata} from 'actions';
import {isCreateModalVisible, getCreateModalForPostId} from 'selectors';

import CreateIssue from './create_issue';

const mapStateToProps = (state) => {
    const postId = getCreateModalForPostId(state);
    const post = getPost(state, postId);

    return {
        visible: isCreateModalVisible(state),
        post,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeCreateModal,
    create: createIssue,
    getMetadata: getCreateIssueMetadata,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssue);
