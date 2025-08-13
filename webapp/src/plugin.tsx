// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Action, Store} from 'redux';

import {isSystemMessage} from 'mattermost-redux/utils/post_utils';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import ConnectModal from 'components/modals/connect_modal';
import DisconnectModal from 'components/modals/disconnect_modal';

import CreateIssuePostMenuAction from 'components/post_menu_actions/create_issue';

import CreateIssueModal from 'components/modals/create_issue';

import ChannelSubscriptionsModal from 'components/modals/channel_subscriptions';

import AttachCommentToIssuePostMenuAction from 'components/post_menu_actions/attach_comment_to_issue';
import AttachCommentToIssueModal from 'components/modals/attach_comment_modal';
import SetupUI from 'components/setup_ui';
import LinkTooltip from 'components/jira_ticket_tooltip';
import {canUserConnect, getInstalledInstances, isUserConnected} from 'selectors';
import {isCombinedUserActivityPost} from 'utils/posts';
import {GlobalState} from 'types/store';

import manifest from './manifest';

import reducers from './reducers';
import {
    getConnected,
    getSettings,
    handleConnectChange,
    handleConnectFlow,
    handleInstanceStatusChange,
    openAttachCommentToIssueModal,
    openCreateModal,
} from './actions';

import Hooks from './hooks/hooks';

const setupUILater = (registry: any, store: Store<object, Action<object>>): () => Promise<void> => async () => {
    registry.registerReducer(reducers);

    const settings = await store.dispatch(getSettings());
    const {id: PluginId} = manifest;

    try {
        await getConnected()(store.dispatch, store.getState);

        if (settings.ui_enabled) {
            registry.registerRootComponent(ConnectModal);
            registry.registerRootComponent(DisconnectModal);
            registry.registerRootComponent(CreateIssueModal);
            registry.registerPostDropdownMenuAction({
                text: CreateIssuePostMenuAction,
                action: (postId: string) => {
                    const state = store.getState() as GlobalState;
                    const userConnected = isUserConnected(state);
                    const userCanConnect = canUserConnect(state);
                    const installedInstances = getInstalledInstances(state);

                    if (!installedInstances.length) {
                        return;
                    }

                    if (userConnected) {
                        store.dispatch<any>(openCreateModal(postId));
                    } else if (userCanConnect) {
                        store.dispatch<any>(handleConnectFlow());
                    }
                },
                filter: (postId: string): boolean => {
                    const state = store.getState() as GlobalState;
                    const post = getPost(state, postId);
                    const oldSystemMessageOrNull = post ? isSystemMessage(post) : true;
                    const systemMessage = isCombinedUserActivityPost(post) || oldSystemMessageOrNull;
                    const installedInstances = getInstalledInstances(state);

                    if (systemMessage || !installedInstances.length) {
                        return false;
                    }

                    return true;
                },
            });
            registry.registerRootComponent(AttachCommentToIssueModal);
            registry.registerPostDropdownMenuAction({
                text: AttachCommentToIssuePostMenuAction,
                action: (postId: string) => {
                    const state = store.getState() as GlobalState;
                    const post = getPost(state, postId);
                    const oldSystemMessageOrNull = post ? isSystemMessage(post) : true;
                    const systemMessage = isCombinedUserActivityPost(post) || oldSystemMessageOrNull;
                    const userConnected = isUserConnected(state);

                    if (systemMessage || !userConnected) {
                        return;
                    }

                    store.dispatch<any>(openAttachCommentToIssueModal(postId));
                },
                filter: (postId: string): boolean => {
                    const state = store.getState() as GlobalState;
                    const post = getPost(state, postId);
                    const oldSystemMessageOrNull = post ? isSystemMessage(post) : true;
                    const systemMessage = isCombinedUserActivityPost(post) || oldSystemMessageOrNull;
                    const userConnected = isUserConnected(state);

                    return !systemMessage && userConnected;
                },
            });
            registry.registerLinkTooltipComponent(LinkTooltip);
        }

        registry.registerRootComponent(ChannelSubscriptionsModal);

        const hooks = new Hooks(store, settings);
        registry.registerSlashCommandWillBePostedHook(hooks.slashCommandWillBePostedHook);
    } finally {
        registry.registerWebSocketEventHandler(`custom_${PluginId}_connect`, handleConnectChange(store));
        registry.registerWebSocketEventHandler(`custom_${PluginId}_disconnect`, handleConnectChange(store));
        registry.registerWebSocketEventHandler(`custom_${PluginId}_update_defaults`, handleConnectChange(store));
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
