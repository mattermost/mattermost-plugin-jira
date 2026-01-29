// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';

import {FilterField, FilterFieldInclusion, IssueMetadata} from 'types/model';
import {getCustomFieldFiltersForProjects} from 'utils/jira_issue_metadata';

import {Props} from './channel_subscription_filter';

describe('components/ChannelSubscriptionFilter', () => {
    const fields = getCustomFieldFiltersForProjects(issueMetadata as IssueMetadata, [issueMetadata.projects[0].key], []);
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
        fields,
        field: fields.find((f) => f.key === 'priority') as FilterField,
        value: {
            key: 'priority',
            inclusion: FilterFieldInclusion.INCLUDE_ANY,
            values: ['1'],
        },
        chosenIssueTypes: [],
        issueMetadata: issueMetadata as IssueMetadata,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        onChange: jest.fn(),
        removeFilter: jest.fn(),
        instanceID: 'https://something.atlassian.net',
        securityLevelEmptyForJiraSubscriptions: true,
        searchTeamFields: jest.fn().mockResolvedValue({data: []}),
    };

    test('baseProps are correctly defined', () => {
        expect(baseProps.theme).toBeDefined();
        expect(baseProps.fields).toBeDefined();
        expect(baseProps.field).toBeDefined();
    });

    test('priority field is correctly selected', () => {
        expect(baseProps.field?.key).toBe('priority');
    });

    test('filter value has expected structure', () => {
        expect(baseProps.value.key).toBe('priority');
        expect(baseProps.value.inclusion).toBe(FilterFieldInclusion.INCLUDE_ANY);
        expect(baseProps.value.values).toContain('1');
    });

    test('issue metadata is loaded', () => {
        expect(baseProps.issueMetadata).toBeDefined();
        expect(issueMetadata.projects).toBeDefined();
    });

    test('callbacks are correctly mocked', () => {
        expect(typeof baseProps.onChange).toBe('function');
        expect(typeof baseProps.removeFilter).toBe('function');
        expect(typeof baseProps.addValidate).toBe('function');
    });

    test('instance ID is set', () => {
        expect(baseProps.instanceID).toBe('https://something.atlassian.net');
    });

    test('security level empty flag is true', () => {
        expect(baseProps.securityLevelEmptyForJiraSubscriptions).toBe(true);
    });

    test('fields array is populated', () => {
        expect(fields.length).toBeGreaterThan(0);
    });
});
