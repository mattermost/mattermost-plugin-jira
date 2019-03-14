// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import CreateIssuePostMenuAction from 'components/post_menu_actions/create_issue';
import CreateIssueModal from 'components/modals/create_issue';

import PluginId from 'plugin_id';

import reducers from './reducers';
import {handleConnect, getConnected} from './actions';

export default class Plugin {
    async initialize(registry, store) {
        registry.registerReducer(reducers);

        await getConnected()(store.dispatch, store.getState);

        registry.registerRootComponent(CreateIssueModal);
        registry.registerPostDropdownMenuComponent(CreateIssuePostMenuAction);

        registry.registerWebSocketEventHandler(`custom_${PluginId}_connect`, handleConnect(store));
    }
}