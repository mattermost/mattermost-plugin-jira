// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import JiraIcon from 'components/icon';

import CreateIssuePostMenuAction from 'components/post_menu_actions/create_issue';
import CreateIssueModal from 'components/modals/create_issue';
import ChannelSettingsModal from 'components/modals/channel_settings';

import AttachCommentToIssuePostMenuAction from 'components/post_menu_actions/attach_comment_to_issue';
import AttachCommentToIssueModal from 'components/modals/attach_comment_to_issue';

import PluginId from 'plugin_id';

import reducers from './reducers';
import {handleConnectChange, getConnected, openChannelSettings, handleInstanceStatusChange} from './actions';
import Hooks from './hooks/hooks';

export default class Plugin {
    async initialize(registry, store) {
        registry.registerReducer(reducers);

        try {
            await getConnected()(store.dispatch, store.getState);

            registry.registerRootComponent(CreateIssueModal);
            registry.registerRootComponent(ChannelSettingsModal);
            registry.registerPostDropdownMenuComponent(CreateIssuePostMenuAction);
            registry.registerChannelHeaderButtonAction(
                <JiraIcon/>,
                (channel) => store.dispatch(openChannelSettings(channel.id)),
                'JIRA',
            );
            registry.registerRootComponent(AttachCommentToIssueModal);
            registry.registerPostDropdownMenuComponent(AttachCommentToIssuePostMenuAction);

            const hooks = new Hooks(store);
            registry.registerSlashCommandWillBePostedHook(hooks.slashCommandWillBePostedHook);
        } catch (err) {
            throw err;
        } finally {
            registry.registerWebSocketEventHandler(`custom_${PluginId}_connect`, handleConnectChange(store));
            registry.registerWebSocketEventHandler(`custom_${PluginId}_disconnect`, handleConnectChange(store));
            registry.registerWebSocketEventHandler(`custom_${PluginId}_instance_status`, handleInstanceStatusChange(store));
        }
    }
}
