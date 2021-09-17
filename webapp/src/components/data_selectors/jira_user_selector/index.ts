// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {searchUsers} from 'actions';

import JiraUserSelector from './jira_user_selector';

const mapDispatchToProps = (dispatch) => bindActionCreators({searchUsers}, dispatch);

export default connect(null, mapDispatchToProps)(JiraUserSelector);
