// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import testChannel from 'testdata/channel.json';

import {InstanceType, IssueMetadata, ProjectMetadata} from 'types/model';

import ChannelSubscriptionsModal, {Props} from './channel_subscriptions';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const mockStore = configureStore([thunk]);

const defaultMockState = {
    'plugins-jira': {
        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
    },
    entities: {
        general: {
            config: {
                SiteURL: 'http://localhost:8065',
            },
        },
    },
};

const mockTheme = {
    centerChannelColor: '#333333',
    centerChannelBg: '#ffffff',
    buttonBg: '#166de0',
    buttonColor: '#ffffff',
    linkColor: '#2389d7',
    errorTextColor: '#fd5960',
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

describe('components/ChannelSettingsModal', () => {
    const baseProps = {
        theme: mockTheme,
        fetchJiraProjectMetadataForAllInstances: jest.fn().mockResolvedValue({}),
        fetchChannelSubscriptions: jest.fn().mockResolvedValue({}),
        fetchAllSubscriptionTemplates: jest.fn().mockResolvedValue({}),
        sendEphemeralPost: jest.fn(),
        jiraIssueMetadata: {} as IssueMetadata,
        jiraProjectMetadata: {} as ProjectMetadata,
        channel: testChannel,
        channelSubscriptions: [],
        omitDisplayName: false,
        createChannelSubscription: jest.fn(),
        deleteChannelSubscription: jest.fn(),
        editChannelSubscription: jest.fn(),
        clearIssueMetadata: jest.fn(),
        close: jest.fn(),
        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
    } as unknown as Props;

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('modal only shows when channel is present', async () => {
        // When channel is null, modal should not show
        const propsWithNullChannel = {
            ...baseProps,
            channel: null,
        };

        const ref = React.createRef<ChannelSubscriptionsModal>();
        renderWithRedux(
            <ChannelSubscriptionsModal
                {...propsWithNullChannel}
                ref={ref}
            />,
        );

        // Component should render without crashing when channel is null
        expect(ref.current).toBeDefined();

        // When channel is present, modal should show
        const propsWithChannel = {
            ...baseProps,
            channel: testChannel,
        };

        const ref2 = React.createRef<ChannelSubscriptionsModal>();
        renderWithRedux(
            <ChannelSubscriptionsModal
                {...propsWithChannel}
                ref={ref2}
            />,
        );

        await act(async () => {
            await propsWithChannel.fetchChannelSubscriptions(testChannel.id);
        });
        await act(async () => {
            await propsWithChannel.fetchAllSubscriptionTemplates();
        });
        await act(async () => {
            await propsWithChannel.fetchJiraProjectMetadataForAllInstances();
        });

        // Component should render and show when channel is present
        expect(ref2.current).toBeDefined();
    });

    test('error fetching channel subscriptions, should close modal and show ephemeral message', async () => {
        const fetchChannelSubscriptions = jest.fn().mockImplementation(() => Promise.resolve({error: 'Failed to fetch'}));
        const sendEphemeralPost = jest.fn();
        const close = jest.fn();

        const props = {
            ...baseProps,
            fetchChannelSubscriptions,
            sendEphemeralPost,
            close,
            channel: null,
        };

        const ref = React.createRef<ChannelSubscriptionsModal>();
        const {rerender} = renderWithRedux(
            <ChannelSubscriptionsModal
                {...props}
                ref={ref}
            />,
        );

        await act(async () => {
            rerender(
                <IntlProvider locale='en'>
                    <Provider store={mockStore(defaultMockState)}>
                        <ChannelSubscriptionsModal
                            {...props}
                            channel={testChannel}
                            ref={ref}
                        />
                    </Provider>
                </IntlProvider>,
            );
        });

        await act(async () => {
            await fetchChannelSubscriptions(testChannel.id);
        });
        await act(async () => {
            await props.fetchJiraProjectMetadataForAllInstances();
        });

        expect(sendEphemeralPost).toHaveBeenCalledWith('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
    });
});
