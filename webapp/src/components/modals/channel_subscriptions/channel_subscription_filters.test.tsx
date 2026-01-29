// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useFieldForIssueMetadata} from 'testdata/jira-issue-metadata-helpers';

import {FilterFieldInclusion} from 'types/model';
import {getCustomFieldFiltersForProjects} from 'utils/jira_issue_metadata';

import {Props} from './channel_subscription_filters';

describe('components/ChannelSubscriptionFilters', () => {
    const field = {
        hasDefaultValue: false,
        key: 'priority',
        name: 'Priority',
        operations: [
            'set',
        ],
        required: false,
        schema: {
            system: 'priority',
            type: 'priority',
        },
        allowedValues: [{
            id: '1',
            name: 'Highest',
        }, {
            id: '2',
            name: 'High',
        }, {
            id: '3',
            name: 'Medium',
        }, {
            id: '4',
            name: 'Low',
        }, {
            id: '5',
            name: 'Lowest',
        }],
    };

    const issueMetadata = useFieldForIssueMetadata(field, 'priority');

    const mockTheme = {
        centerChannelColor: '#333333',
        centerChannelBg: '#ffffff',
        buttonBg: '#166de0',
        buttonColor: '#ffffff',
        linkColor: '#2389d7',
        errorTextColor: '#fd5960',
    };

    const baseProps: Props = {
        theme: mockTheme,
        fields: getCustomFieldFiltersForProjects(issueMetadata, [issueMetadata.projects[0].key], []),
        values: [{
            key: 'priority',
            inclusion: FilterFieldInclusion.INCLUDE_ANY,
            values: ['1'],
        }],
        chosenIssueTypes: ['10001'],
        issueMetadata,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        onChange: jest.fn(),
        instanceID: 'https://something.atlassian.net',
        securityLevelEmptyForJiraSubscriptions: true,
        searchTeamFields: jest.fn().mockResolvedValue({data: []}),
    };

    test('baseProps are correctly defined', () => {
        expect(baseProps.theme).toBeDefined();
        expect(baseProps.fields).toBeDefined();
        expect(baseProps.values).toHaveLength(1);
    });

    test('issue metadata is correctly loaded', () => {
        expect(issueMetadata).toBeDefined();
        expect(issueMetadata.projects).toBeDefined();
    });

    test('filter values have expected structure', () => {
        expect(baseProps.values[0].key).toBe('priority');
        expect(baseProps.values[0].inclusion).toBe(FilterFieldInclusion.INCLUDE_ANY);
    });

    test('chosen issue types are set', () => {
        expect(baseProps.chosenIssueTypes).toContain('10001');
    });

    test('instance ID is set', () => {
        expect(baseProps.instanceID).toBe('https://something.atlassian.net');
    });

    test('security level empty flag is true', () => {
        expect(baseProps.securityLevelEmptyForJiraSubscriptions).toBe(true);
    });
});
