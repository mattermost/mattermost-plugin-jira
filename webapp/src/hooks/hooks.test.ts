// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    openChannelSettings,
    openCreateModalWithoutPost,
    openDisconnectModal,
    sendEphemeralPost,
} from 'actions';
import {instanceIsInstalled, isUserConnected} from 'selectors';

import Hooks from './hooks';

jest.mock('actions', () => ({
    getConnected: jest.fn(() => ({type: 'GET_CONNECTED'})),
    handleConnectFlow: jest.fn(),
    openChannelSettings: jest.fn(() => ({type: 'OPEN_CHANNEL_SETTINGS'})),
    openCreateModalWithoutPost: jest.fn(() => ({type: 'OPEN_CREATE_MODAL'})),
    openDisconnectModal: jest.fn(() => ({type: 'OPEN_DISCONNECT_MODAL'})),
    sendEphemeralPost: jest.fn(() => ({type: 'EPHEMERAL'})),
}));

jest.mock('selectors', () => ({
    getPluginSettings: jest.fn(() => ({ui_enabled: true})),
    getUserConnectedInstances: jest.fn(() => []),
    instanceIsInstalled: jest.fn(),
    isUserConnected: jest.fn(),
}));

describe('Hooks slash commands', () => {
    const contextArgs = {channel_id: 'channel-id'};
    const store = {getState: jest.fn(() => ({})), dispatch: jest.fn(() => Promise.resolve())};

    beforeEach(() => {
        jest.clearAllMocks();
        (instanceIsInstalled as jest.Mock).mockReturnValue(true);
        (isUserConnected as jest.Mock).mockReturnValue(true);
    });

    test.each([
        ['/jira create', openCreateModalWithoutPost],
        ['/jira subscribe', openChannelSettings],
    ])('%s opens its modal for a connected user', async (command, action) => {
        const result = await new Hooks(store, {}).slashCommandWillBePostedHook(command, contextArgs);

        expect(result).toEqual({});
        expect(action).toHaveBeenCalled();
    });

    test.each([
        ['/jira create', openCreateModalWithoutPost],
        ['/jira subscribe', openChannelSettings],
    ])('%s does not open its modal for an unconnected user', async (command, action) => {
        (isUserConnected as jest.Mock).mockReturnValue(false);

        const result = await new Hooks(store, {}).slashCommandWillBePostedHook(command, contextArgs);

        expect(result).toEqual({});
        expect(sendEphemeralPost).toHaveBeenCalledWith(expect.stringContaining('not connected to Jira'));
        expect(action).not.toHaveBeenCalled();
    });

    test.each([
        ['/jira create', openCreateModalWithoutPost],
        ['/jira subscribe', openChannelSettings],
        ['/jira disconnect', openDisconnectModal],
    ])('%s stops when no instance is installed', async (command, action) => {
        (instanceIsInstalled as jest.Mock).mockReturnValue(false);

        const result = await new Hooks(store, {}).slashCommandWillBePostedHook(command, contextArgs);

        expect(result).toEqual({});
        expect(sendEphemeralPost).toHaveBeenCalledWith(expect.stringContaining('no Jira instance installed'));
        expect(action).not.toHaveBeenCalled();
    });
});
