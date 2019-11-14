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
        fetchChannelSubscriptions: jest.fn(),
        fetchJiraProjectMetadata: jest.fn(),
        sendEphemeralPost: jest.fn(),
        jiraIssueMetadata: {} as IssueMetadata,
        jiraProjectMetadata: {} as ProjectMetadata,
        channel: null,
        channelSubscriptions: [],
        omitDisplayName: false,
        createChannelSubscription: jest.fn(),
        deleteChannelSubscription: jest.fn(),
        editChannelSubscription: jest.fn(),
        clearIssueMetadata: jest.fn(),
        close: () => jest.fn(),
    } as Props;

    test('modal only shows when channel, channelSubscriptions, and jiraProjectMetadata props are present', async () => {
        const props = {
            ...baseProps,
            channel: null,
            channelSubscriptions: [],
            jiraProjectMetadata: {} as ProjectMetadata,
        };

        const wrapper = shallow<ChannelSettingsModal>(
            <ChannelSettingsModal {...props}/>
        );

        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(0);
        expect(wrapper.find(FullScreenModal).props().show).toBe(false);

        wrapper.setProps({
            ...props,
            channel: testChannel,
            channelSubscriptions: null,
            jiraProjectMetadata: {} as ProjectMetadata,
        });

        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(0);
        expect(wrapper.find(FullScreenModal).props().show).toBe(false);

        wrapper.setProps({
            ...props,
            channel: testChannel,
            channelSubscriptions: [],
            jiraProjectMetadata: null,
        });

        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(0);
        expect(wrapper.find(FullScreenModal).props().show).toBe(false);

        wrapper.setProps({
            ...props,
            channel: testChannel,
            channelSubscriptions: [],
            jiraProjectMetadata: {} as ProjectMetadata,
        });

        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(1);
        expect(wrapper.find(FullScreenModal).props().show).toBe(true);
    });

    test('error fetching channel subscriptions, should close modal and show ephemeral message', async () => {
        const props = {
            ...baseProps,
            fetchChannelSubscriptions: jest.fn().mockImplementation(() => Promise.resolve({error: 'Failed to fetch'})),
            fetchJiraProjectMetadata: jest.fn().mockImplementation(() => Promise.resolve({data: {}})),
            sendEphemeralPost: jest.fn(),
            close: jest.fn(),
        };

        const wrapper = shallow<ChannelSettingsModal>(
            <ChannelSettingsModal {...props}/>
        );

        wrapper.setProps({...props, channel: testChannel});

        expect(props.fetchChannelSubscriptions).toHaveBeenCalled();

        await Promise.resolve();

        expect(props.close).toHaveBeenCalled();
        expect(props.sendEphemeralPost).toHaveBeenCalledWith('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
    });

    test('error fetching project metadata, should close modal and show ephemeral message', async () => {
        const props = {
            ...baseProps,
            fetchChannelSubscriptions: jest.fn().mockImplementation(() => Promise.resolve({data: []})),
            fetchJiraProjectMetadata: jest.fn().mockImplementation(() => Promise.resolve({error: 'Failed to fetch'})),
            sendEphemeralPost: jest.fn(),
            close: jest.fn(),
        };

        const wrapper = shallow<ChannelSettingsModal>(
            <ChannelSettingsModal {...props}/>
        );

        wrapper.setProps({...props, channel: testChannel});

        expect(props.fetchJiraProjectMetadata).toHaveBeenCalled();

        await Promise.resolve();
        await Promise.resolve();

        expect(props.close).toHaveBeenCalled();
        expect(props.sendEphemeralPost).toHaveBeenCalledWith('Failed to get Jira project information. Please contact your Mattermost administrator.');
    });
});
