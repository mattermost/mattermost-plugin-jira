// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import DisconnectModalForm from './disconnect_modal_form';

const mockStore = configureStore([thunk]);

const defaultMockState = {
    'plugins-jira': {
        installedInstances: [],
        connectedInstances: [],
    },
    entities: {
        general: {
            config: {
                SiteURL: 'http://localhost:8065',
            },
        },
    },
};

const renderWithRedux = (ui: React.ReactElement, initialState = defaultMockState) => {
    const store = mockStore(initialState);
    return {
        store,
        ...render(
            <IntlProvider locale='en'>
                <Provider store={store}>{ui}</Provider>
            </IntlProvider>,
        ),
    };
};

describe('components/DisconnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        sendEphemeralPost: jest.fn().mockResolvedValue({}),
        disconnectUser: jest.fn().mockResolvedValue({}),
    };

    const mockTheme = {
        centerChannelColor: '#333333',
        centerChannelBg: '#ffffff',
        buttonBg: '#166de0',
        buttonColor: '#ffffff',
        linkColor: '#2389d7',
        errorTextColor: '#fd5960',
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
