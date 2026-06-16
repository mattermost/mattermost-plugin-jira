// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';

import {searchIssues} from 'actions';

import JiraIssueSelector from './jira_issue_selector';

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators({
    searchIssues,
}, dispatch);

export default connect(null, mapDispatchToProps)(JiraIssueSelector);
