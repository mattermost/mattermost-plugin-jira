// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {Dispatch, bindActionCreators} from 'redux';

import {getTheme} from 'mattermost-redux/selectors/entities/preferences';

import {getConnected} from '../../../actions';
import {getInstalledInstances, getPluginServerRoute} from '../../../selectors';

import {GlobalState} from 'types/store';

import RHSStatusSetting, {Props} from './rhs_status_setting';

type RHSStatusDispatchProps = Pick<Props, 'getConnected'>;

const mapStateToProps = (state: GlobalState) => {
    return {
        installedInstances: getInstalledInstances(state),
        theme: getTheme(state),
        pluginServerRoute: getPluginServerRoute(state),
    };
};

const rhsStatusDispatch = {
    getConnected,
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators<typeof rhsStatusDispatch, RHSStatusDispatchProps>(rhsStatusDispatch, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(RHSStatusSetting);
