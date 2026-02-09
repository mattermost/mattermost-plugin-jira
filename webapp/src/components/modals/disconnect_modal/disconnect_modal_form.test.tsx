// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act} from '@testing-library/react';

import {mockTheme, renderWithRedux} from 'testlib/test-utils';

import DisconnectModalForm from './disconnect_modal_form';

describe('components/DisconnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        sendEphemeralPost: jest.fn().mockResolvedValue({}),
        disconnectUser: jest.fn().mockResolvedValue({}),
    };

    const baseProps = {
        ...baseActions,
        visible: true,
        theme: mockTheme,
        connectedInstances: [],
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should render component', async () => {
        const props = {...baseProps};
        const ref = React.createRef<DisconnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <DisconnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        expect(ref.current).toBeDefined();
    });

    test('should close modal and send ephemeral post on submit success', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const sendEphemeralPost = jest.fn().mockResolvedValue({});
        const disconnectUser = jest.fn().mockResolvedValue({});

        const props = {
            ...baseProps,
            closeModal,
            sendEphemeralPost,
            disconnectUser,
        };
        const ref = React.createRef<DisconnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <DisconnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            ref.current?.handleInstanceChoice('', 'https://something.atlassian.net');
        });
        expect(ref.current?.state.selectedInstance).toEqual('https://something.atlassian.net');

        await act(async () => {
            ref.current?.submit({preventDefault: jest.fn()});
        });
        await act(async () => {
            await Promise.resolve();
        });

        expect(disconnectUser).toHaveBeenCalledWith('https://something.atlassian.net');
        expect(sendEphemeralPost).toHaveBeenCalledWith('Successfully disconnected from Jira instance https://something.atlassian.net');
        expect(closeModal).toHaveBeenCalled();
    });

    test('should show error on submit fail', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const sendEphemeralPost = jest.fn().mockResolvedValue({});
        const disconnectUser = jest.fn().mockResolvedValue({error: 'Error disconnecting'});

        const props = {
            ...baseProps,
            closeModal,
            sendEphemeralPost,
            disconnectUser,
        };
        const ref = React.createRef<DisconnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <DisconnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            ref.current?.handleInstanceChoice('', 'https://something.atlassian.net');
        });
        expect(ref.current?.state.selectedInstance).toEqual('https://something.atlassian.net');

        await act(async () => {
            ref.current?.submit({preventDefault: jest.fn()});
        });
        await act(async () => {
            await Promise.resolve();
        });

        expect(disconnectUser).toHaveBeenCalled();
        expect(sendEphemeralPost).not.toHaveBeenCalled();
        expect(closeModal).not.toHaveBeenCalled();

        expect(ref.current?.state.error).toEqual('Error disconnecting');
    });
});
