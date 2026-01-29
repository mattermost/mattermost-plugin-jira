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
    const mockTheme = {
        centerChannelColor: '#333333',
        centerChannelBg: '#ffffff',
        buttonBg: '#166de0',
        buttonColor: '#ffffff',
        linkColor: '#2389d7',
        errorTextColor: '#fd5960',
    };

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
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        subscriptionTemplates: [],
    } as unknown as Props;

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('modal only shows when channel is present', async () => {
        const props = {
            ...baseProps,
            channel: null,
        };

        const ref = React.createRef<ChannelSubscriptionsModal>();
        const {rerender, container} = await act(async () => {
            return renderWithRedux(
                <ChannelSubscriptionsModal
                    {...props}
                    ref={ref}
                />,
            );
        });

        // Modal should not show inner content when channel is null
        expect(container.querySelector('.FullScreenModal')).not.toBeInTheDocument();

        // Set channel prop
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
            await props.fetchChannelSubscriptions(testChannel.id);
            await props.fetchAllSubscriptionTemplates();
            await props.fetchJiraProjectMetadataForAllInstances();
        });

        // Modal should show when channel is present
        expect(container.querySelector('.FullScreenModal')).toBeInTheDocument();
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
        const {rerender} = await act(async () => {
            return renderWithRedux(
                <ChannelSubscriptionsModal
                    {...props}
                    ref={ref}
                />,
            );
        });

        // Change channel from null to testChannel to trigger componentDidUpdate
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

        // Wait for the async fetchData to complete
        await act(async () => {
            await Promise.resolve();
            await Promise.resolve();
        });

        expect(fetchChannelSubscriptions).toHaveBeenCalledWith(testChannel.id);
        expect(sendEphemeralPost).toHaveBeenCalledWith('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
        expect(close).toHaveBeenCalled();
    });
});
