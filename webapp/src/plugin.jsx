// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import CreateIssuePostMenuAction from 'components/post_menu_actions/create_issue';
import CreateIssueModal from 'components/modals/create_issue';

import reducers from './reducers';

export default class Plugin {
    initialize(registry, store) {
        // @see https://developers.mattermost.com/extend/plugins/webapp/reference/
        registry.registerReducer(reducers);

        registry.registerRootComponent(CreateIssueModal);
        registry.registerPostDropdownMenuComponent(CreateIssuePostMenuAction);
    }
}