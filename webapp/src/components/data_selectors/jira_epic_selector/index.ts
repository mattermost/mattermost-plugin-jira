// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {searchIssues} from 'actions';

import JiraEpicSelector from './jira_epic_selector';

const mapDispatchToProps = (dispatch) => bindActionCreators({
    searchIssues,
}, dispatch);

export default connect(null, mapDispatchToProps)(JiraEpicSelector);
