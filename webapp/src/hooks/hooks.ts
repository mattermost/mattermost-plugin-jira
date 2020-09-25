// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {openCreateModalWithoutPost, openChannelSettings, sendEphemeralPost, openDisconnectModal, handleConnectFlow, getConnected} from '../actions';
import {isUserConnected, getInstalledInstances, getPluginSettings, getUserConnectedInstances} from '../selectors';

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
    }

    handleCreateSlashCommand = (message: string, contextArgs: ContextArgs) => {
        if (!getInstalledInstances(this.store.getState())) {
            this.store.dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
            return Promise.resolve({});
        }
        if (!isUserConnected(this.store.getState())) {
            this.store.dispatch(sendEphemeralPost('Your Mattermost account is not connected to Jira. Please use `/jira connect` to connect your account, then try again.'));
            return Promise.resolve({});
        }
        let description = '';
        if (message.startsWith(createCommand)) {
            description = message.slice(createCommand.length).trim();
        } else if (message.startsWith(issueCreateCommand)) {
            description = message.slice(issueCreateCommand.length).trim();
        }
        this.store.dispatch(openCreateModalWithoutPost(description, contextArgs.channel_id));
        return Promise.resolve({});
    }

    handleSubscribeSlashCommand = (message: string, contextArgs: ContextArgs) => {
        if (!getInstalledInstances(this.store.getState()).length) {
            this.store.dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
            return Promise.resolve({});
        }
        if (!isUserConnected(this.store.getState())) {
            this.store.dispatch(sendEphemeralPost('Your Mattermost account is not connected to Jira. Please use `/jira connect` to connect your account, then try again.'));
            return Promise.resolve({});
        }

        this.store.dispatch(getConnected());
        this.store.dispatch(openChannelSettings(contextArgs.channel_id));
        return Promise.resolve({});
    }

    handleDisconnectSlashCommand = (message: string, contextArgs: ContextArgs) => {
        const state = this.store.getState();
        const instances = getInstalledInstances(state);
        const connectedInstances = getUserConnectedInstances(state);

        if (!instances.length) {
            this.store.dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
            return Promise.resolve({});
        }

        let args = '';
        if (message.startsWith(disconnectCommand)) {
            args = message.slice(disconnectCommand.length).trim();
        } else if (message.startsWith(instanceDisconnectCommand)) {
            args = message.slice(instanceDisconnectCommand.length).trim();
        }
        if (connectedInstances.length < 2 || args) {
            // Let the server take care of the command
            return Promise.resolve({message, args: contextArgs});
        }

        this.store.dispatch(openDisconnectModal());
        return Promise.resolve({});
    }

    handleConnectSlashCommand = (message: string, contextArgs: ContextArgs) => {
        let instanceID = '';
        if (message.startsWith(connectCommand)) {
            instanceID = message.slice(connectCommand.length).trim();
        } else if (message.startsWith(instanceConnectCommand)) {
            instanceID = message.slice(instanceConnectCommand.length).trim();
        }
        this.store.dispatch(handleConnectFlow(instanceID));
        return Promise.resolve({});
    }
}
