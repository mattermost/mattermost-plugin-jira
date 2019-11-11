// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import testChannel from 'testdata/channel.json';

import ChannelSettingsModal, {Props} from './channel_settings';

describe('components/ChannelSettingsModal', () => {
    const baseProps = {
        sendEphemeralPost: jest.fn(),
        close: jest.fn(),
        fetchJiraProjectMetadata: jest.fn(),
        fetchChannelSubscriptions: jest.fn(),
    } as Props;

    test('error fetching project metadata, should close modal and show ephemeral message', async () => {
        const props = {
            ...baseProps,
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

        expect(props.close).toHaveBeenCalled();
        expect(props.sendEphemeralPost).toHaveBeenCalledWith('Failed to get Jira project information. Please contact your Mattermost administrator.');
    });
});
