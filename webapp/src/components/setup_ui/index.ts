// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';

import {GlobalState} from 'mattermost-redux/types/store';
import {GenericAction} from 'mattermost-redux/types/actions';

import {isUserConnected, isInstanceInstalled} from 'selectors';
import {openChannelSettings} from 'actions';

import SetupUI from './setup_ui';

const mapStateToProps = (state: GlobalState): object => {
    return {
        userConnected: isUserConnected(state),
        instanceInstalled: isInstanceInstalled(state),
    };
};

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>): object => bindActionCreators({
    openChannelSettings,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(SetupUI);
