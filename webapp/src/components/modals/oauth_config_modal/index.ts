// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {isOAuthConfigModalVisible} from 'selectors';

import {openOAuthConfigModal, closeOAuthConfigModal, configureCloudOAuthInstance, handleInstallOAuthFlow} from 'actions';

import OAuthConfigModal from './oauth_config_modal';

const mapStateToProps = (state) => {
    return {
        visible: isOAuthConfigModalVisible(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    open: openOAuthConfigModal,
    closeModal: closeOAuthConfigModal,
    configure: configureCloudOAuthInstance,
    handleInstallOAuthFlow,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(OAuthConfigModal);
