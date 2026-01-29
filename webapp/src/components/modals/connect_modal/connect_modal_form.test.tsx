// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import {InstanceType} from 'types/model';

import ConnectModalForm from './connect_modal_form';

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

describe('components/ConnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        redirectConnect: jest.fn().mockResolvedValue({}),
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
        connectedInstances: [
            {
                instance_id: 'https://something.atlassian.net',
                type: InstanceType.CLOUD,
            },
        ],
        installedInstances: [
            {
                instance_id: 'https://something.atlassian.net',
                type: InstanceType.CLOUD,
            },
            {
                instance_id: 'http://localhost:8080',
                type: InstanceType.SERVER,
            },
        ],
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', async () => {
        const props = {...baseProps};
        const ref = React.createRef<ConnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <ConnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        expect(ref.current).toBeDefined();
    });

    test('should redirect on submit', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const redirectConnect = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            closeModal,
            redirectConnect,
        };
        const ref = React.createRef<ConnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <ConnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            ref.current?.handleInstanceChoice('', 'http://localhost:8080');
        });
        expect(ref.current?.state.selectedInstance).toEqual('http://localhost:8080');
        expect(ref.current?.state.error).toEqual('');
    });

    test('should show error when user is already connected to instance', async () => {
        const props = {...baseProps};
        const ref = React.createRef<ConnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <ConnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            ref.current?.handleInstanceChoice('', 'https://something.atlassian.net');
        });
        expect(ref.current?.state.selectedInstance).toEqual('https://something.atlassian.net');
        expect(ref.current?.state.error).toEqual('You are already connected to this Jira instance.');
    });
});
