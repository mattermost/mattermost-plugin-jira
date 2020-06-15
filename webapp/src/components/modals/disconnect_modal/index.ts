// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {closeDisconnectModal, disconnectUser, sendEphemeralPost} from 'actions';
import {isDisconnectModalVisible, getUserConnectedInstances} from 'selectors';

import DisconnectModal from './disconnect_modal';

const mapStateToProps = (state) => {
    return {
        connectedInstances: getUserConnectedInstances(state),
        visible: isDisconnectModalVisible(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    closeModal: closeDisconnectModal,
    disconnectUser,
    sendEphemeralPost,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(DisconnectModal);
