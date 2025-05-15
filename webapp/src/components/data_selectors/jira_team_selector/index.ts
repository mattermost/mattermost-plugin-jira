// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {searchTeamFields} from 'actions';

import JiraTeamSelector from './jira_team_selector';

const mapDispatchToProps = (dispatch) => bindActionCreators({searchTeamFields}, dispatch);

export default connect(null, mapDispatchToProps)(JiraTeamSelector);
