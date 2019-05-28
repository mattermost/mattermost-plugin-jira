// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import {openAttachCommentToIssueModal} from 'actions';

import {getCurrentUserLocale, isConnected} from 'selectors';

import AttachCommentToIssuePostMenuAction from './attach_comment_to_issue';

const mapStateToProps = (state, ownProps) => {
    const post = getPost(state, ownProps.postId);
    return {
        locale: getCurrentUserLocale(state),
        isSystemMessage: isSystemMessage(post),
        connected: isConnected(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    open: openAttachCommentToIssueModal,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssuePostMenuAction);
