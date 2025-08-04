// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {searchAutoCompleteFields} from '../../../actions';

import JiraAutoCompleteSelector from './jira_autocomplete_selector';

const mapDispatchToProps = (dispatch) => bindActionCreators({searchAutoCompleteFields}, dispatch);

export default connect(null, mapDispatchToProps)(JiraAutoCompleteSelector);
