// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import testChannel from 'testdata/channel.json';

import {IssueMetadata, ProjectMetadata} from 'types/model';

import FullScreenModal from '../full_screen_modal/full_screen_modal';

import ChannelSettingsModal, {Props} from './channel_settings';
import ChannelSettingsModalInner from './channel_settings_internal';

describe('components/ChannelSettingsModal', () => {
    const baseProps = {
        theme: {},
        fetchJiraIssueMetadataForProjects: jest.fn(),
        fetchChannelSubscriptions: jest.fn().mockResolvedValue({}),
        fetchJiraProjectMetadata: jest.fn().mockResolvedValue({}),
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
        close: () => jest.fn(),
    } as Props;

    test('modal only shows when channel is present', async () => {
        const props = {
            ...baseProps,
            channel: null,
        };

        const wrapper = shallow<ChannelSettingsModal>(
            <ChannelSettingsModal {...props}/>
        );

        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(0);
        expect(wrapper.find(FullScreenModal).props().show).toBe(false);

        wrapper.setProps({
            ...props,
            channel: testChannel,
        });

        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(1);
        expect(wrapper.find(FullScreenModal).props().show).toBe(true);
    });

    test('error fetching channel subscriptions, should close modal and show ephemeral message', async () => {
        const props = {
            ...baseProps,
            fetchChannelSubscriptions: jest.fn().mockImplementation(() => Promise.resolve({error: 'Failed to fetch'})),
            sendEphemeralPost: jest.fn(),
            close: jest.fn(),
        };

        const wrapper = shallow<ChannelSettingsModalInner>(
            <ChannelSettingsModalInner {...props}/>
        );

        wrapper.setProps({...props, channel: testChannel});

        expect(props.fetchChannelSubscriptions).toHaveBeenCalled();

        await Promise.resolve();

        expect(props.close).toHaveBeenCalled();
        expect(props.sendEphemeralPost).toHaveBeenCalledWith('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
    });
});
