// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {fetchJiraProjectMetadata} from 'actions';

import {
    getJiraProjectMetadata,
    getInstalledInstances,
    getUserConnectedInstances,
    getDefaultUserInstance,
} from 'selectors';

import JiraInstanceAndProjectSelector from './jira_instance_and_project_selector';

const mapStateToProps = (state) => {
    const installedInstances = getInstalledInstances(state);
    const connectedInstances = getUserConnectedInstances(state);
    const defaultUserInstance = getDefaultUserInstance(state);

    return {
        installedInstances,
        connectedInstances,
        defaultUserInstance,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    fetchJiraProjectMetadata,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(JiraInstanceAndProjectSelector);
