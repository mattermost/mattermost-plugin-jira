// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {isDesktopApp, isMinimumDesktopAppVersion} from '../utils/user_agent';
import {openCreateModalWithoutPost, openChannelSettings, sendEphemeralPost} from '../actions';
import {isUserConnected, getInstalledInstances, getPluginSettings, getDefaultConnectInstance, getUserConnectedInstances} from '../selectors';
import PluginId from 'plugin_id';

export default class Hooks {
    constructor(store, settings) {
        this.store = store;
        this.settings = settings;
    }

    slashCommandWillBePostedHook = (rawMessage, contextArgs) => {
        let message;
        if (rawMessage) {
            message = rawMessage.trim();
        }

        const pluginSettings = getPluginSettings(this.store.getState());

        let shouldEnableCreate = false;
        if (pluginSettings) {
            shouldEnableCreate = pluginSettings.ui_enabled;
        } else if (this.settings) {
            shouldEnableCreate = this.settings.ui_enabled;
        }

        const createCommand = '/jira create';
        const connectCommand = '/jira connect';
        const subscribeCommand = '/jira subscribe';
        if (message && message.startsWith(createCommand) && shouldEnableCreate) {
            if (!getInstalledInstances(this.store.getState())) {
                this.store.dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
                return Promise.resolve({});
            }
            if (!isUserConnected(this.store.getState())) {
                this.store.dispatch(sendEphemeralPost('Your Mattermost account is not connected to Jira. Please use `/jira connect` to connect your account, then try again.'));
                return Promise.resolve({});
            }
            const description = message.slice(createCommand.length).trim();
            this.store.dispatch(openCreateModalWithoutPost(description, contextArgs.channel_id));
            return Promise.resolve({});
        }

        if (message && message.startsWith(connectCommand)) {
            if (!getInstalledInstances(this.store.getState())) {
                this.store.dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
                return Promise.resolve({});
            }

            const args = message.slice(connectCommand.length).trim();
            let instance = getDefaultConnectInstance(this.store.getState());

            if (args) {
                const instanceID = args;
                const connectedInstances = getUserConnectedInstances(this.store.getState());
                const alreadyConnected = connectedInstances[instanceID];

                if (alreadyConnected) {
                    this.store.dispatch(sendEphemeralPost(
                        'Your Jira account at ' + alreadyConnected.InstanceID + ' is already linked to your Mattermost account. Please use `/jira disconnect` to disconnect.'));
                    return Promise.resolve({});
                }

                const instances = getInstalledInstances(this.store.getState());
                instance = instances[instanceID];
                if (!instance) {
                    const errMsg = 'Jira instance ' + instanceID + ' is not installed. Please type `/jira instance list` to see the available Jira instances.';
                    this.store.dispatch(sendEphemeralPost(errMsg));
                    return Promise.resolve({});
                }
            }

            if (instance && instance.type === 'server' && isDesktopApp() && !isMinimumDesktopAppVersion(4, 3, 0)) { // eslint-disable-line no-magic-numbers
                const errMsg = 'Your version of the Mattermost desktop client does not support authenticating between Jira and Mattermost directly. To connect your Jira account with Mattermost, please go to Mattermost via your web browser and type `/jira connect`, or [check the Mattermost download page](https://mattermost.com/download/#mattermostApps) to get the latest version of the desktop client.';
                this.store.dispatch(sendEphemeralPost(errMsg));
                return Promise.resolve({});
            }

            if (instance && instance.instance_id) {
                const encodedID = btoa(instance.instance_id);
                const target = '/plugins/' + PluginId + '/instance/' + encodedID + '/user/connect';
                window.open(target, '_blank');
            } else {
                // TODO: <><> present instance picker to choose an installed instance
            }
            return Promise.resolve({});
        }

        if (message && message === subscribeCommand) {
            // TODO: <><> add instance picker and/or filter to Subscribe UI
            if (!getInstalledInstances(this.store.getState())) {
                this.store.dispatch(sendEphemeralPost('There is no Jira instance installed. Please contact your system administrator.'));
                return Promise.resolve({});
            }
            if (!isUserConnected(this.store.getState())) {
                this.store.dispatch(sendEphemeralPost('Your Mattermost account is not connected to Jira. Please use `/jira connect` to connect your account, then try again.'));
                return Promise.resolve({});
            }

            this.store.dispatch(openChannelSettings(contextArgs.channel_id));
            return Promise.resolve({});
        }

        return Promise.resolve({message, args: contextArgs});
    }
}
