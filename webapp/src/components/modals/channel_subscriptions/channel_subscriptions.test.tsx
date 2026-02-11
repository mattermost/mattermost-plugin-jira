// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act} from '@testing-library/react';

import testChannel from 'testdata/channel.json';

import {InstanceType, IssueMetadata, ProjectMetadata} from 'types/model';
import {mockTheme, renderWithRedux} from 'testlib/test-utils';

import ChannelSubscriptionsModal, {Props} from './channel_subscriptions';

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

        expect(ref.current).toBeDefined();

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
            channel: testChannel,
        };

        const ref = React.createRef<ChannelSubscriptionsModal>();
        renderWithRedux(
            <ChannelSubscriptionsModal
                {...props}
                ref={ref}
            />,
        );

        await act(async () => {
            await ref.current?.fetchData();
        });

        expect(sendEphemeralPost).toHaveBeenCalledWith('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
    });
});
