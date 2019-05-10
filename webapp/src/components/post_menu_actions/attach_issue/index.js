// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import {openAttachModal} from 'actions';

import {getCurrentUserLocale} from 'selectors';

import PluginId from 'plugin_id';

import AttachIssuePostMenuAction from './attach_issue';

const mapStateToProps = (state, ownProps) => {
    const post = getPost(state, ownProps.postId);
    return {
        locale: getCurrentUserLocale(state),
        isSystemMessage: isSystemMessage(post),
        connected: state[`plugins-${PluginId}`],
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    open: openAttachModal,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachIssuePostMenuAction);
