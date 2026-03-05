// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getSprintByID, searchSprints} from 'actions';

import JiraSprintSelector from './jira_sprint_selector';

const mapDispatchToProps = (dispatch) => bindActionCreators({searchSprints, getSprintByID}, dispatch);

export default connect(null, mapDispatchToProps)(JiraSprintSelector);
