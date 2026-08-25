// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';

import {isUserConnected} from 'selectors';
import {openChannelSettings} from 'actions';

import type {GlobalState} from 'types/store';

import SetupUI from './setup_ui';

const mapStateToProps = (state: GlobalState) => {
    return {
        userConnected: isUserConnected(state),
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators({
    openChannelSettings,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(SetupUI);
