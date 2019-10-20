// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {getPluginServerRoute} from 'selectors';

import JiraEpicSelector from './jira_epic_selector';

const mapStateToProps = (state) => {
    return {
        fetchEpicsEndpoint: getPluginServerRoute(state) + '/api/v2/get-search-epics',
    };
};

export default connect(mapStateToProps)(JiraEpicSelector);
