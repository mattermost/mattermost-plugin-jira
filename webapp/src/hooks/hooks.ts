// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    getConnected,
    handleConnectFlow,
    openChannelSettings,
    openCreateModalWithoutPost,
    openDisconnectModal,
    sendEphemeralPost,
} from 'actions';
import {
    getPluginSettings,
    getUserConnectedInstances,
    instanceIsInstalled,
    isUserConnected,
} from 'selectors';

type ContextArgs = {channel_id: string};

const createCommand = '/jira create';
const connectCommand = '/jira connect';
const disconnectCommand = '/jira disconnect';
const issueCreateCommand = '/jira issue create';
const instanceConnectCommand = '/jira instance connect';
const instanceDisconnectCommand = '/jira instance disconnect';
const subscribeCommand = '/jira subscribe';
const subscribeEditCommand = '/jira subscribe edit';

export default class Hooks {
    private store: any;
    private settings: any;

    constructor(store: any, settings: any) {
        this.store = store;
        this.settings = settings;
    }

    slashCommandWillBePostedHook = (rawMessage: string, contextArgs: ContextArgs) => {
        let message;
        if (rawMessage) {
            message = rawMessage.trim();
        }

        if (!message) {
            return Promise.resolve({message, args: contextArgs});
        }

        const pluginSettings = getPluginSettings(this.store.getState());

        let shouldEnableCreate = false;
        if (pluginSettings) {
            shouldEnableCreate = pluginSettings.ui_enabled;
        } else if (this.settings) {
            shouldEnableCreate = this.settings.ui_enabled;
        }

        if ((message.startsWith(createCommand) || message.startsWith(issueCreateCommand)) && shouldEnableCreate) {
            return this.handleCreateSlashCommand(message, contextArgs);
        }

        if (message.startsWith(connectCommand) || message.startsWith(instanceConnectCommand)) {
            return this.handleConnectSlashCommand(message, contextArgs);
        }

        if (message.startsWith(disconnectCommand) || message.startsWith(instanceDisconnectCommand)) {
            return this.handleDisconnectSlashCommand(message, contextArgs);
        }

        if (message === subscribeCommand || message === subscribeEditCommand) {
            return this.handleSubscribeSlashCommand(message, contextArgs);
        }

        return Promise.resolve({message, args: contextArgs});
    };

    handleCreateSlashCommand = async (message: string, contextArgs: ContextArgs) => {
        if (!(await this.checkInstanceIsInstalled())) {
            return {};
        }
        if (!(await this.checkUserIsConnected())) {
            return {};
        }

        let description = '';
        if (message.startsWith(createCommand)) {
            description = message.slice(createCommand.length).trim();
        } else if (message.startsWith(issueCreateCommand)) {
            description = message.slice(issueCreateCommand.length).trim();
        }
        this.store.dispatch(openCreateModalWithoutPost(description, contextArgs.channel_id));
        return {};
    };

    handleSubscribeSlashCommand = async (message: string, contextArgs: ContextArgs) => {
        if (!(await this.checkInstanceIsInstalled())) {
            return {};
        }
        if (!(await this.checkUserIsConnected())) {
            return {};
        }

        this.store.dispatch(getConnected());
        this.store.dispatch(openChannelSettings(contextArgs.channel_id));
        return {};
    };

    handleDisconnectSlashCommand = async (message: string, contextArgs: ContextArgs) => {
        if (!(await this.checkInstanceIsInstalled())) {
            return {};
        }

        const connectedInstances = getUserConnectedInstances(this.store.getState());
        let args = '';
        if (message.startsWith(disconnectCommand)) {
            args = message.slice(disconnectCommand.length).trim();
        } else if (message.startsWith(instanceDisconnectCommand)) {
            args = message.slice(instanceDisconnectCommand.length).trim();
        }
        if (connectedInstances.length < 2 || args) {
            // Let the server take care of the command
            return {message, args: contextArgs};
        }

        this.store.dispatch(openDisconnectModal());
        return {};
    };

    checkInstanceIsInstalled = async (): Promise<boolean> => {
        if (!instanceIsInstalled(this.store.getState())) {
            await this.store.dispatch(getConnected());
            if (!instanceIsInstalled(this.store.getState())) {
                this.store.dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
                return false;
            }
        }

        return true;
    };

    checkUserIsConnected = async (): Promise<boolean> => {
        if (!isUserConnected(this.store.getState())) {
            await this.store.dispatch(getConnected());
            if (!isUserConnected(this.store.getState())) {
                this.store.dispatch(sendEphemeralPost('Your Mattermost account is not connected to Jira. Please use `/jira connect` to connect your account, then try again.'));
                return false;
            }
        }

        return true;
    };

    handleConnectSlashCommand = (message: string, contextArgs: ContextArgs) => {
        let instanceID = '';
        if (message.startsWith(connectCommand)) {
            instanceID = message.slice(connectCommand.length).trim();
        } else if (message.startsWith(instanceConnectCommand)) {
            instanceID = message.slice(instanceConnectCommand.length).trim();
        }
        this.store.dispatch(handleConnectFlow(instanceID));
        return Promise.resolve({});
    };
}
