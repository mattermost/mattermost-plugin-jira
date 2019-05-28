// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import {openCreateModal} from 'actions';

import {getCurrentUserLocale, isUserConnected, isInstanceInstalled} from 'selectors';

import CreateIssuePostMenuAction from './create_issue';

const mapStateToProps = (state, ownProps) => {
    const post = getPost(state, ownProps.postId);
    return {
        locale: getCurrentUserLocale(state),
        isSystemMessage: isSystemMessage(post),
        userConnected: isUserConnected(state),
        instanceInstalled: isInstanceInstalled(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    open: openCreateModal,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssuePostMenuAction);
