// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {Dispatch, bindActionCreators} from 'redux';

import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import {
    getConnected,
    handleConnectFlow,
    loadMoreRHSIssues,
    openCreateModalWithoutPost,
    resolveAndFetchRHSIssues,
    restoreRHSViewState,
} from 'actions';
import {
    getConnectedCloudInstances,
    getRHSError,
    getRHSInstanceID,
    getRHSIsLast,
    getRHSIssues,
    getRHSLoading,
    getRHSSort,
    getRHSTab,
    getRHSTabs,
} from 'selectors';

import {GlobalState} from 'types/store';

import Rhs, {Props} from './rhs';

type RHSDispatchProps = Pick<Props, 'getConnected' | 'handleConnectFlow' | 'loadMoreRHSIssues' | 'openCreateModalWithoutPost' | 'resolveAndFetchRHSIssues' | 'restoreRHSViewState'>;

const mapStateToProps = (state: GlobalState) => {
    return {
        connectedCloud: getConnectedCloudInstances(state),
        channelId: getCurrentChannelId(state) || '',
        instanceID: getRHSInstanceID(state),
        tab: getRHSTab(state),
        sort: getRHSSort(state),
        issues: getRHSIssues(state),
        tabs: getRHSTabs(state),
        isLast: getRHSIsLast(state),
        loading: getRHSLoading(state),
        error: getRHSError(state),
    };
};

const rhsDispatch = {
    getConnected,
    handleConnectFlow,
    loadMoreRHSIssues,
    openCreateModalWithoutPost,
    resolveAndFetchRHSIssues,
    restoreRHSViewState,
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators<typeof rhsDispatch, RHSDispatchProps>(rhsDispatch, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(Rhs);
