// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {closeConnectModal, redirectConnect} from 'actions';
import {isConnectModalVisible, getUserConnectedInstances} from 'selectors';

import ConnectModal from './connect_modal';

const mapStateToProps = (state) => {
    return {
        visible: isConnectModalVisible(state),
        connectedInstances: getUserConnectedInstances(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    closeModal: closeConnectModal,
    redirectConnect,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(ConnectModal);
