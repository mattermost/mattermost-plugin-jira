// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import {openChannelSettings} from 'actions';

import {isUserConnected, isInstanceInstalled} from 'selectors';

import ChannelHeaderMenuAction from './channel_header_menu';

const mapStateToProps = (state) => {
    return {
        channelId: getCurrentChannelId(state),
        userConnected: isUserConnected(state),
        isInstanceInstalled: isInstanceInstalled(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    open: openChannelSettings,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(ChannelHeaderMenuAction);
