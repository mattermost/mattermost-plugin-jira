// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import {handleConnectFlow, openCreateModal} from 'actions';

import {
    canUserConnect,
    getCurrentUserLocale,
    getInstalledInstances,
    isUserConnected,
} from 'selectors';
import {isCombinedUserActivityPost} from 'utils/posts';

import CreateIssuePostMenuAction from './create_issue';

const mapStateToProps = (state, ownProps) => {
    const post = getPost(state, ownProps.postId);
    const oldSystemMessageOrNull = post ? isSystemMessage(post) : true;
    const systemMessage = isCombinedUserActivityPost(post) || oldSystemMessageOrNull;

    return {
        locale: getCurrentUserLocale(state),
        isSystemMessage: systemMessage,
        userConnected: isUserConnected(state),
        userCanConnect: canUserConnect(state),
        installedInstances: getInstalledInstances(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    open: openCreateModal,
    handleConnectFlow,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssuePostMenuAction);
