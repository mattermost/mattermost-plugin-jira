// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {checkPermissionsForIssue, removePermissionsBlockFromPost} from '../../actions';

import PostTypeRestrictedPermissions from './post_type_restricted_permissions';

const mapDispatchToProps = (dispatch) => bindActionCreators({
    checkPermissions: checkPermissionsForIssue,
    removePermissions: removePermissionsBlockFromPost,
}, dispatch);

export default connect(null, mapDispatchToProps)(PostTypeRestrictedPermissions);
