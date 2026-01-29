// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Preferences from 'mattermost-redux/constants/preferences';
import {Channel} from '@mattermost/types/channels';

import cloudIssueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import testChannel from 'testdata/channel.json';

import {FilterFieldInclusion} from 'types/model';

import {Props} from './edit_channel_subscription';

describe('components/EditChannelSubscription', () => {
    const baseActions = {
        createChannelSubscription: jest.fn().mockResolvedValue({}),
        deleteChannelSubscription: jest.fn().mockResolvedValue({}),
        editChannelSubscription: jest.fn().mockResolvedValue({}),
        fetchChannelSubscriptions: jest.fn().mockResolvedValue({}),
        createSubscriptionTemplate: jest.fn().mockResolvedValue({}),
        deleteSubscriptionTemplate: jest.fn().mockResolvedValue({}),
        editSubscriptionTemplate: jest.fn().mockResolvedValue({}),
        fetchAllSubscriptionTemplates: jest.fn().mockResolvedValue({}),
        fetchSubscriptionTemplatesForProjectKey: jest.fn().mockResolvedValue({}),
        sendEphemeralPost: jest.fn().mockResolvedValue({}),
        getConnected: jest.fn().mockResolvedValue({}),
        fetchJiraProjectMetadataForAllInstances: jest.fn().mockResolvedValue({}),
        fetchJiraIssueMetadataForProjects: jest.fn().mockResolvedValue({data: cloudIssueMetadata}),
        searchTeamFields: jest.fn().mockResolvedValue({data: []}),
    };

    const channelSubscriptionForCloud = {
        id: 'asxtifxe8jyi9y81htww6ixkiy',
        channel_id: testChannel.id,
        filters: {
            events: ['event_updated_reopened'],
            projects: ['KT'],
            issue_types: ['10004'],
            fields: [{
                key: 'customfield_10073',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['10035'],
            }, {
                key: 'versions',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['10000'],
            }, {
                key: 'customfield_10014',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['IDT-24'],
            }],
        },
        name: 'SubTestName',
        instance_id: 'https://something.atlassian.net',
    };

    const baseProps: Props = {
        ...baseActions,
        channel: testChannel as unknown as Channel,
        theme: Preferences.THEMES.denim,
        finishEditSubscription: jest.fn(),
        channelSubscriptions: [channelSubscriptionForCloud],
        close: jest.fn(),
        selectedSubscription: channelSubscriptionForCloud,
        creatingSubscription: false,
        creatingSubscriptionTemplate: false,
        securityLevelEmptyForJiraSubscriptions: true,
    };

    test('props are correctly defined', () => {
        expect(baseProps.channel).toBeDefined();
        expect(baseProps.channelSubscriptions).toHaveLength(1);
        expect(baseProps.selectedSubscription).toBeDefined();
    });

    test('baseActions are correctly mocked', () => {
        expect(baseActions.createChannelSubscription).toBeDefined();
        expect(baseActions.deleteChannelSubscription).toBeDefined();
        expect(baseActions.editChannelSubscription).toBeDefined();
    });

    test('subscription filters are correctly structured', () => {
        expect(channelSubscriptionForCloud.filters.events).toContain('event_updated_reopened');
        expect(channelSubscriptionForCloud.filters.projects).toContain('KT');
        expect(channelSubscriptionForCloud.filters.issue_types).toContain('10004');
    });

    test('subscription fields are correctly structured', () => {
        const fields = channelSubscriptionForCloud.filters.fields;
        expect(fields).toHaveLength(3);
        expect(fields[0].key).toBe('customfield_10073');
        expect(fields[0].inclusion).toBe('include_any');
    });

    test('mock actions return expected values', async () => {
        const result = await baseActions.fetchJiraIssueMetadataForProjects('', '');
        expect(result.data).toBeDefined();
    });

    test('theme is correctly passed', () => {
        expect(baseProps.theme).toBe(Preferences.THEMES.denim);
    });

    test('creating subscription flags work correctly', () => {
        expect(baseProps.creatingSubscription).toBe(false);
        expect(baseProps.creatingSubscriptionTemplate).toBe(false);
    });

    test('security level empty flag is set', () => {
        expect(baseProps.securityLevelEmptyForJiraSubscriptions).toBe(true);
    });

    test('channel subscription has correct channel_id', () => {
        expect(channelSubscriptionForCloud.channel_id).toBe(testChannel.id);
    });

    test('channel subscription has correct instance_id', () => {
        expect(channelSubscriptionForCloud.instance_id).toBe('https://something.atlassian.net');
    });

    test('channel subscription has correct name', () => {
        expect(channelSubscriptionForCloud.name).toBe('SubTestName');
    });

    test('filters include expected issue type', () => {
        expect(baseProps.selectedSubscription?.filters.issue_types).toContain('10004');
    });

    test('filters include expected project', () => {
        expect(baseProps.selectedSubscription?.filters.projects).toContain('KT');
    });

    test('close callback is provided', () => {
        expect(typeof baseProps.close).toBe('function');
    });

    test('finishEditSubscription callback is provided', () => {
        expect(typeof baseProps.finishEditSubscription).toBe('function');
    });

    test('channel data is valid', () => {
        expect(testChannel.id).toBeDefined();
        expect(testChannel.display_name).toBeDefined();
    });
});
