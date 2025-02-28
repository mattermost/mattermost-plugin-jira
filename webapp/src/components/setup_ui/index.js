// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {isUserConnected} from 'selectors';
import {openChannelSettings} from 'actions';

import SetupUI from './setup_ui';

const mapStateToProps = (state) => {
    return {
        userConnected: isUserConnected(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    openChannelSettings,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(SetupUI);
