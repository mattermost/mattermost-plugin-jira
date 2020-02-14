// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {searchLabels} from 'actions';

import JiraLabelSelector from './jira_label_selector';

const mapDispatchToProps = (dispatch) => bindActionCreators({searchLabels}, dispatch);

export default connect(null, mapDispatchToProps)(JiraLabelSelector);
