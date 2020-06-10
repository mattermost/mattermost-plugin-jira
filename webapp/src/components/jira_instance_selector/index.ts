// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getConnected} from 'actions';
import {getInstalledInstances, getUserConnectedInstances} from 'selectors';

import JiraInstanceSelector from './jira_instance_selector';

const mapStateToProps = (state) => {
    const instances = getInstalledInstances(state);
    const connectedInstances = getUserConnectedInstances(state);
    return {
        instances,
        connectedInstances,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    getConnected,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(JiraInstanceSelector);
