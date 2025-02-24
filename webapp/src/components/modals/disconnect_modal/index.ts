// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {Dispatch, bindActionCreators} from 'redux';

import {closeDisconnectModal, disconnectUser, sendEphemeralPost} from 'actions';
import {getUserConnectedInstances, isDisconnectModalVisible} from 'selectors';

import {GlobalState} from 'types/store';

import DisconnectModal from './disconnect_modal';

const mapStateToProps = (state: GlobalState) => {
    return {
        connectedInstances: getUserConnectedInstances(state),
        visible: isDisconnectModalVisible(state),
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators({
    closeModal: closeDisconnectModal,
    disconnectUser,
    sendEphemeralPost,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(DisconnectModal);
