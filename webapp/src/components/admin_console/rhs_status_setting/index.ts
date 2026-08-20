// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {Dispatch, bindActionCreators} from 'redux';

import {getTheme} from 'mattermost-redux/selectors/entities/preferences';

import {fetchRHSStatuses, getConnected} from '../../../actions';
import {getInstalledInstances} from '../../../selectors';

import {GlobalState} from 'types/store';

import RHSStatusSetting, {Props} from './rhs_status_setting';

const mapStateToProps = (state: GlobalState) => {
    return {
        installedInstances: getInstalledInstances(state),
        theme: getTheme(state),
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators({
    fetchRHSStatuses,
    getConnected,
}, dispatch) as unknown as Pick<Props, 'fetchRHSStatuses' | 'getConnected'>;

export default connect(mapStateToProps, mapDispatchToProps)(RHSStatusSetting);
