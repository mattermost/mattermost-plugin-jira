// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import CreateIssuePostMenuAction from 'components/post_menu_actions/create_issue';
import CreateIssueModal from 'components/modals/create_issue';

import PluginId from 'plugin_id';

import reducers from './reducers';
import {handleConnectChange, getConnected} from './actions';

export default class Plugin {
    async initialize(registry, store) {
        registry.registerReducer(reducers);

        try {
            await getConnected()(store.dispatch, store.getState);

            registry.registerRootComponent(CreateIssueModal);
            registry.registerPostDropdownMenuComponent(CreateIssuePostMenuAction);
        } catch (err) {
            throw err;
        } finally {
            registry.registerWebSocketEventHandler(`custom_${PluginId}_connect`, handleConnectChange(store));
            registry.registerWebSocketEventHandler(`custom_${PluginId}_disconnect`, handleConnectChange(store));
        }
    }
}
