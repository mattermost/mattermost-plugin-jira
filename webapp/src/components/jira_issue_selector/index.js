// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {getPluginServerRoute} from '../../selectors';

import JiraIssueSelector from './jira_issue_selector';

const mapStateToProps = (state) => {
    return {
        fetchIssuesEndpoint: getPluginServerRoute(state) + '/api/v2/get-search-issues',
    };
};

export default connect(mapStateToProps)(JiraIssueSelector);
