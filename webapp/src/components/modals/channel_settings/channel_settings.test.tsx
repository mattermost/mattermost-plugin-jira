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
        jiraIssueMetadata: {} as IssueMetadata,
        fetchJiraIssueMetadataForProjects: jest.fn(),
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

    test('modal only shows when channel is passed in', async () => {
        const props = {
            ...baseProps,
            channel: null,
        };

        const wrapper = shallow<ChannelSettingsModal>(
            <ChannelSettingsModal {...props}/>
        );

        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(0);
        expect(wrapper.find(FullScreenModal).props().show).toBe(false);

        wrapper.setProps({...props, channel: testChannel});
        expect(wrapper.find(ChannelSettingsModalInner).length).toEqual(1);
        expect(wrapper.find(FullScreenModal).props().show).toBe(true);
    });
});
