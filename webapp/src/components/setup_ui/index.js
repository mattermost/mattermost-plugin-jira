// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {isUserConnected, isInstanceInstalled} from 'selectors';
import {openChannelSettings} from 'actions';

import SetupUI from './setup_ui';

const mapStateToProps = (state) => {
    return {
        userConnected: isUserConnected(state),
        instanceInstalled: isInstanceInstalled(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    openChannelSettings,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(SetupUI);
