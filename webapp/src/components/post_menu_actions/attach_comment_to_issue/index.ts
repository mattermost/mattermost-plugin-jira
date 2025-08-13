// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {GlobalState} from 'types/store';
import {getCurrentUserLocale, isUserConnected} from 'selectors';

import AttachCommentToIssuePostMenuAction from './attach_comment_to_issue';

function mapStateToProps(state: GlobalState): {actionText: string} {
    const locale = getCurrentUserLocale(state);
    const userConnected = isUserConnected(state);

    if (!userConnected) {
        return {actionText: ''};
    }

    let actionText;
    switch (locale) {
    case 'es':
        actionText = 'Adjuntar a incidencia de Jira';
        break;
    default:
        actionText = 'Attach to Jira Issue';
    }

    return {actionText};
}

export default connect(mapStateToProps)(AttachCommentToIssuePostMenuAction);
