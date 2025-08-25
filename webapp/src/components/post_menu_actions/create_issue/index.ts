// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {GlobalState} from 'types/store';

import {canUserConnect, getCurrentUserLocale, isUserConnected} from 'selectors';

import CreateIssuePostMenuAction from './create_issue';

function mapStateToProps(state: GlobalState): {actionText: string} {
    const locale = getCurrentUserLocale(state);
    const userConnected: boolean = isUserConnected(state);
    const userCanConnect: boolean = canUserConnect(state);

    let actionText;
    if (userConnected) {
        switch (locale) {
        case 'es':
            actionText = 'Crear incidencia en Jira';
            break;
        default:
            actionText = 'Create Jira Issue';
        }
    } else if (userCanConnect) {
        actionText = 'Connect to Jira';
    } else {
        actionText = '';
    }

    return {
        actionText,
    };
}

export default connect(mapStateToProps)(CreateIssuePostMenuAction);