// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Store, Action} from 'redux';

import ConnectModal from 'components/modals/connect_modal';
import DisconnectModal from 'components/modals/disconnect_modal';

import CreateIssuePostMenuAction from 'components/post_menu_actions/create_issue';
import CreateIssueModal from 'components/modals/create_issue';
import ChannelSettingsModal from 'components/modals/channel_settings';

import AttachCommentToIssuePostMenuAction from 'components/post_menu_actions/attach_comment_to_issue';
import AttachCommentToIssueModal from 'components/modals/attach_comment_to_issue';
import SetupUI from 'components/setup_ui';

import PluginId from 'plugin_id';

import reducers from './reducers';
import {handleConnectChange, getConnected, handleInstanceStatusChange, getSettings} from './actions';
import Hooks from './hooks/hooks';

const setupUILater = (registry: any, store: Store<object, Action<object>>): () => Promise<void> => async () => {
    registry.registerReducer(reducers);

    const settings = await store.dispatch(getSettings());

    try {
        await getConnected()(store.dispatch, store.getState);

        if (settings.ui_enabled) {
            registry.registerRootComponent(ConnectModal);
            registry.registerRootComponent(DisconnectModal);
            registry.registerRootComponent(CreateIssueModal);
            registry.registerPostDropdownMenuComponent(CreateIssuePostMenuAction);
            registry.registerRootComponent(AttachCommentToIssueModal);
            registry.registerPostDropdownMenuComponent(AttachCommentToIssuePostMenuAction);
        }

        registry.registerRootComponent(ChannelSettingsModal);

        const hooks = new Hooks(store, settings);
        registry.registerSlashCommandWillBePostedHook(hooks.slashCommandWillBePostedHook);
    } finally {
        registry.registerWebSocketEventHandler(`custom_${PluginId}_connect`, handleConnectChange(store));
        registry.registerWebSocketEventHandler(`custom_${PluginId}_disconnect`, handleConnectChange(store));
        registry.registerWebSocketEventHandler(`custom_${PluginId}_instance_status`, handleInstanceStatusChange(store));
    }
};

export default class Plugin {
    private haveSetupUI = false;
    private headerButtonId = '';
    private setupUI?: () => Promise<void>;

    private finishedSetupUI = () => {
        this.haveSetupUI = true;
    };

    private setHeaderButtonId = (id: string) => {
        this.headerButtonId = id;
    };

    public async initialize(registry: PluginRegistry, store: Store<object, Action<object>>) {
        this.setupUI = setupUILater(registry, store);
        this.haveSetupUI = false;
        this.headerButtonId = '';

        // Register the dummy component, which will call setupUI when it is activated (i.e., when the user logs in)
        registry.registerRootComponent(
            () => {
                return (
                    <SetupUI
                        registry={registry}
                        setupUI={this.setupUI}
                        haveSetupUI={this.haveSetupUI}
                        finishedSetupUI={this.finishedSetupUI}
                        headerButtonId={this.headerButtonId}
                        setHeaderButtonId={this.setHeaderButtonId}
                    />
                );
            });
    }
}
