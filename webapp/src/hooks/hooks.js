// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {openCreateModalWithoutPost, sendEphemeralPost} from '../actions';
import {isUserConnected, isInstanceInstalled} from '../selectors';
import PluginId from 'plugin_id';

export default class Hooks {
    constructor(store) {
        this.store = store;
    }

    slashCommandWillBePostedHook = (message, contextArgs) => {
        if (message && (message.startsWith('/jira create ') || message === '/jira create')) {
            if (!isInstanceInstalled(this.store.getState())) {
                sendEphemeralPost(this.store, 'There is no Jira instance installed. Please contact your system administrator.');
                return Promise.resolve({});
            }
            if (!isUserConnected(this.store.getState())) {
                sendEphemeralPost(this.store, 'Your Mattermost account is not connected to Jira. Please use `/jira connect` to connect your account, then try again.');
                return Promise.resolve({});
            }
            const description = message.slice(12).trim();
            this.store.dispatch(openCreateModalWithoutPost(description, contextArgs.channel_id));
            return Promise.resolve({});
        }

        if (message && (message.startsWith('/jira connect') || message === '/jira connect')) {
            if (!isInstanceInstalled(this.store.getState())) {
                sendEphemeralPost(this.store, 'There is no Jira instance installed. Please contact your system administrator.');
                return Promise.resolve({});
            }
            if (isUserConnected(this.store.getState())) {
                sendEphemeralPost(this.store, 'You already have a Jira account linked to your Mattermost account. Please use `/jira disconnect` to disconnect.');
                return Promise.resolve({});
            }
            window.open('/plugins/' + PluginId + '/user/connect', '_blank');
            return Promise.resolve({});
        }

        return Promise.resolve({message, args: contextArgs});
    }
}
