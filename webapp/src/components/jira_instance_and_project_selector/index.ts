// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {fetchJiraProjectMetadata, getConnected} from 'actions';

import {getDefaultUserInstanceID, getInstalledInstances, getUserConnectedInstances} from 'selectors';

import JiraInstanceAndProjectSelector from './jira_instance_and_project_selector';

const mapStateToProps = (state) => {
    const installedInstances = getInstalledInstances(state);
    const connectedInstances = getUserConnectedInstances(state);
    const defaultUserInstanceID = getDefaultUserInstanceID(state);

    return {
        installedInstances,
        connectedInstances,
        defaultUserInstanceID,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    fetchJiraProjectMetadata,
    getConnected,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(JiraInstanceAndProjectSelector);
