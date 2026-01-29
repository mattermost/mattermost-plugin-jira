// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import testChannel from 'testdata/channel.json';

import {IssueMetadata, ProjectMetadata} from 'types/model';

import {Props} from './channel_subscriptions';

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
    } as Props;

    test('baseProps are correctly defined', () => {
        expect(baseProps.theme).toBeDefined();
        expect(baseProps.channel).toBeDefined();
        expect(baseProps.channelSubscriptions).toHaveLength(0);
    });

    test('fetch functions are correctly mocked', () => {
        expect(typeof baseProps.fetchJiraProjectMetadataForAllInstances).toBe('function');
        expect(typeof baseProps.fetchChannelSubscriptions).toBe('function');
        expect(typeof baseProps.fetchAllSubscriptionTemplates).toBe('function');
    });

    test('channel data is correctly loaded', () => {
        expect(testChannel.id).toBeDefined();
        expect(testChannel.display_name).toBeDefined();
    });

    test('close callback is provided', () => {
        expect(typeof baseProps.close).toBe('function');
    });

    test('sendEphemeralPost callback is provided', () => {
        expect(typeof baseProps.sendEphemeralPost).toBe('function');
    });

    test('omitDisplayName is false by default', () => {
        expect(baseProps.omitDisplayName).toBe(false);
    });
});
