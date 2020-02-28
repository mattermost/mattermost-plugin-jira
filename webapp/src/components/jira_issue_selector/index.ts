// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';

import {GenericAction} from 'mattermost-redux/types/actions';

import {searchIssues} from 'actions';

import JiraIssueSelector from './jira_issue_selector';

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>): object => bindActionCreators({
    searchIssues,
}, dispatch);

export default connect(null, mapDispatchToProps, null, {withRef: true})(JiraIssueSelector);
