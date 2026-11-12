// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import testChannel from 'testdata/channel.json';

import {InstanceType, IssueMetadata, ProjectMetadata} from 'types/model';
import {renderWithRedux} from 'testlib/test-utils';

import ChannelSubscriptionsModal, {Props} from './channel_subscriptions';

describe('components/ChannelSettingsModal', () => {
    const baseProps = {
        fetchJiraProjectMetadataForAllInstances: jest.fn().mockResolvedValue({data: []}),
        fetchChannelSubscriptions: jest.fn().mockResolvedValue({}),
        fetchAllSubscriptionTemplates: jest.fn().mockResolvedValue({}),
        fetchJiraProjectMetadata: jest.fn().mockResolvedValue({}),
        getConnected: jest.fn().mockResolvedValue({}),
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
            await propsWithChannel.fetchAllSubscriptionTemplates(testChannel.id);
        });
        await act(async () => {
            await propsWithChannel.fetchJiraProjectMetadataForAllInstances();
        });

        expect(ref2.current).toBeDefined();
    });

    const openModal = async (props: Props) => {
        const ui = (channel: Props['channel']) => (
            <ChannelSubscriptionsModal
                {...props}
                subscriptionTemplates={[]}
                channel={channel}
            />
        );
        const result = renderWithRedux(ui(null));
        await act(async () => {
            result.rerender(ui(testChannel as unknown as Props['channel']));
        });
        return result;
    };

    test('should switch between subscriptions and templates tabs', async () => {
        await openModal(baseProps);

        expect(screen.getByRole('dialog', {name: 'Jira Subscriptions'})).toBeInTheDocument();
        expect(screen.getByRole('tabpanel')).toHaveTextContent('No subscriptions yet');
        expect(screen.getByRole('button', {name: 'Create Subscription'})).toBeInTheDocument();

        await userEvent.click(screen.getByRole('tab', {name: 'Templates'}));

        expect(screen.getByRole('tabpanel')).toHaveTextContent('No templates yet');
        expect(screen.getByRole('button', {name: 'Create Template'})).toBeInTheDocument();
    });

    test('should go back from an untouched new subscription without a discard prompt', async () => {
        await openModal(baseProps);

        await userEvent.click(screen.getByRole('button', {name: 'Create Subscription'}));
        expect(screen.getByRole('dialog', {name: 'New Jira Subscription'})).toBeInTheDocument();

        await userEvent.click(screen.getByRole('button', {name: 'Back'}));

        expect(screen.queryByRole('dialog', {name: 'Discard'})).not.toBeInTheDocument();
        expect(screen.getByRole('dialog', {name: 'Jira Subscriptions'})).toBeInTheDocument();
    });

    test('should close when the backdrop is clicked', async () => {
        const close = jest.fn();
        await openModal({...baseProps, close});

        await userEvent.click(screen.getByRole('dialog').parentElement!);

        expect(close).toHaveBeenCalledTimes(1);
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
