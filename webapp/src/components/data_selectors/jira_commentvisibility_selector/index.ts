// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {searchCommentVisibilityFields} from 'actions';

import JiraCommentVisibilitySelector from './jira_commentvisibility_selector';

const mapDispatchToProps = (dispatch) => bindActionCreators({searchCommentVisibilityFields}, dispatch);

export default connect(null, mapDispatchToProps)(JiraCommentVisibilitySelector);
