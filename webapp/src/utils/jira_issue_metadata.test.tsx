// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import createMeta from 'testdata/cloud-get-create-issue-metadata-for-project-many-fields.json';
import {useFieldForIssueMetadata} from 'testdata/jira-issue-metadata-helpers';

import {getCustomFieldFiltersForProjects, getConflictingFields} from './jira_issue_metadata';

describe('components/ChannelSettingsInner', () => {
    test('should return a list of fields', () => {
        const projectKey = createMeta.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(createMeta, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBeGreaterThan(0);
    });

    test('should return blank list if there are no available values', () => {
        const field = {
            hasDefaultValue: false,
            key: 'customfield_10021',
            name: 'Sprint',
            operations: [
                'set',
            ],
            required: false,
            schema: {
                custom: 'com.pyxis.greenhopper.jira:gh-sprint',
                customId: 10021,
                items: 'string',
                type: 'array',
            },
        };

        const metadata = useFieldForIssueMetadata(field, 'customfield_10021');
        const projectKey = metadata.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(metadata, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBe(0);
    });

    test('should return options for multi-select options', () => {
        const field = {
            allowedValues: [
                {
                    id: '10033',
                    self: 'https://mmtest.atlassian.net/rest/api/2/customFieldOption/10033',
                    value: '1',
                },
                {
                    id: '10034',
                    self: 'https://mmtest.atlassian.net/rest/api/2/customFieldOption/10034',
                    value: '2',
                },
            ],
            hasDefaultValue: false,
            key: 'custom1',
            name: 'MJK - Checkbox',
            operations: ['add', 'set', 'remove'],
            required: false,
            schema: {
                custom: 'com.atlassian.jira.plugin.system.customfieldtypes:multicheckboxes',
                customId: 10068,
                items: 'option',
                type: 'array',
            },
        };

        const metadata = useFieldForIssueMetadata(field, 'custom1');
        const projectKey = metadata.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(metadata, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBe(1);

        expect(actual[0].key).toEqual('custom1');
        expect(actual[0].name).toEqual('MJK - Checkbox');
        expect(actual[0].values).toEqual([{value: '10033', label: '1'}, {value: '10034', label: '2'}]);
    });

    test('should return options for single-select options', () => {
        const field = {
            allowedValues: [
                {
                    id: '10035',
                    self: 'https://mmtest.atlassian.net/rest/api/2/customFieldOption/10035',
                    value: '1',
                },
                {
                    id: '10036',
                    self: 'https://mmtest.atlassian.net/rest/api/2/customFieldOption/10036',
                    value: '2',
                },
            ],
            hasDefaultValue: false,
            key: 'custom1',
            name: 'MJK - Radio Buttons',
            operations: [
                'set',
            ],
            required: false,
            schema: {
                custom: 'com.atlassian.jira.plugin.system.customfieldtypes:radiobuttons',
                customId: 10073,
                type: 'option',
            },
        };

        const metadata = useFieldForIssueMetadata(field, 'custom1');
        const projectKey = metadata.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(metadata, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBe(1);

        expect(actual[0].key).toEqual('custom1');
        expect(actual[0].name).toEqual('MJK - Radio Buttons');
        expect(actual[0].values).toEqual([{value: '10035', label: '1'}, {value: '10036', label: '2'}]);
    });

    test('should return options for priority', () => {
        const field = {
            allowedValues: [
                {
                    iconUrl: 'https://mmtest.atlassian.net/images/icons/priorities/highest.svg',
                    id: '1',
                    name: 'Highest',
                    self: 'https://mmtest.atlassian.net/rest/api/2/priority/1',
                },
                {
                    iconUrl: 'https://mmtest.atlassian.net/images/icons/priorities/high.svg',
                    id: '2',
                    name: 'High',
                    self: 'https://mmtest.atlassian.net/rest/api/2/priority/2',
                },
                {
                    iconUrl: 'https://mmtest.atlassian.net/images/icons/priorities/medium.svg',
                    id: '3',
                    name: 'Medium',
                    self: 'https://mmtest.atlassian.net/rest/api/2/priority/3',
                },
                {
                    iconUrl: 'https://mmtest.atlassian.net/images/icons/priorities/low.svg',
                    id: '4',
                    name: 'Low',
                    self: 'https://mmtest.atlassian.net/rest/api/2/priority/4',
                },
                {
                    iconUrl: 'https://mmtest.atlassian.net/images/icons/priorities/lowest.svg',
                    id: '5',
                    name: 'Lowest',
                    self: 'https://mmtest.atlassian.net/rest/api/2/priority/5',
                },
            ],
            defaultValue: {
                iconUrl: 'https://mmtest.atlassian.net/images/icons/priorities/medium.svg',
                id: '3',
                name: 'Medium',
                self: 'https://mmtest.atlassian.net/rest/api/2/priority/3',
            },
            hasDefaultValue: true,
            key: 'priority',
            name: 'Priority',
            operations: [
                'set',
            ],
            required: true,
            schema: {
                system: 'priority',
                type: 'priority',
            },
        };

        const metadata = useFieldForIssueMetadata(field, 'priority');
        const projectKey = metadata.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(metadata, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBe(1);

        expect(actual[0].key).toEqual('priority');
        expect(actual[0].name).toEqual('Priority');
        expect(actual[0].values).toEqual([{value: '1', label: 'Highest'}, {value: '2', label: 'High'}, {value: '3', label: 'Medium'}, {value: '4', label: 'Low'}, {value: '5', label: 'Lowest'}]);
    });

    test('should return options for fix version', () => {
        const field = {
            allowedValues: [{
                archived: false,
                id: '10000',
                name: '5.14 (August 2019)',
                projectId: '10008',
                released: false,
                self: 'https://mmtest.atlassian.net/rest/api/2/version/10000',
            }],
            hasDefaultValue: false,
            key: 'fixVersions',
            name: 'Fix versions',
            operations: ['set', 'add', 'remove'],
            required: false,
            schema: {
                items: 'version',
                system: 'fixVersions',
                type: 'array',
            },
        };

        const metadata = useFieldForIssueMetadata(field, 'fixVersions');
        const projectKey = metadata.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(metadata, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBe(1);

        expect(actual[0].key).toEqual('fixVersions');
        expect(actual[0].name).toEqual('Fix versions');
        expect(actual[0].values).toEqual([{value: '10000', label: '5.14 (August 2019)'}]);
    });

    test('should return options for security level', () => {
        const field = {
            allowedValues: [
                {
                    description: '',
                    id: '10001',
                    name: 'Admin only',
                    self: 'https://mmtest.atlassian.net/rest/api/2/securitylevel/10001',
                },
                {
                    description: '',
                    id: '10000',
                    name: 'Everyone',
                    self: 'https://mmtest.atlassian.net/rest/api/2/securitylevel/10000',
                },
                {
                    description: 'Test staff level',
                    id: '10002',
                    name: 'Staff',
                    self: 'https://mmtest.atlassian.net/rest/api/2/securitylevel/10002',
                },
            ],
            defaultValue: {
                description: '',
                id: '10000',
                name: 'Everyone',
                self: 'https://mmtest.atlassian.net/rest/api/2/securitylevel/10000',
            },
            hasDefaultValue: true,
            key: 'security',
            name: 'Security Level',
            operations: [
                'set',
            ],
            required: false,
            schema: {
                system: 'security',
                type: 'securitylevel',
            },
        };

        const metadata = useFieldForIssueMetadata(field, 'security');
        const projectKey = metadata.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(metadata, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBe(1);

        expect(actual[0].key).toEqual('security');
        expect(actual[0].name).toEqual('Security Level');
        expect(actual[0].values).toEqual([{value: '10001', label: 'Admin only'}, {value: '10000', label: 'Everyone'}, {value: '10002', label: 'Staff'}]);
    });

    test('should return options with a `userDefined` flag for array of strings', () => {
        const field = {
            autoCompleteUrl: 'https://mmtest.atlassian.net/rest/api/1.0/labels/suggest?customFieldId=10071&query=',
            hasDefaultValue: false,
            key: 'custom1',
            name: 'MJK - Labels',
            operations: [
                'add',
                'set',
                'remove',
            ],
            required: false,
            schema: {
                custom: 'com.atlassian.jira.plugin.system.customfieldtypes:labels',
                customId: 10071,
                items: 'string',
                type: 'array',
            },
        };

        const metadata = useFieldForIssueMetadata(field, 'custom1');
        const projectKey = metadata.projects[0].key;

        const actual = getCustomFieldFiltersForProjects(metadata, [projectKey]);
        expect(actual).not.toBe(null);
        expect(actual.length).toBe(1);

        expect(actual[0].key).toEqual('custom1');
        expect(actual[0].name).toEqual('MJK - Labels');
        expect(actual[0].userDefined).toEqual(true);
    });

    test('getConflictingFields should return a list of fields with conflicts', () => {
        let field;
        field = {
            autoCompleteUrl: 'https://mmtest.atlassian.net/rest/api/1.0/labels/suggest?customFieldId=10071&query=',
            hasDefaultValue: false,
            key: 'custom1',
            name: 'MJK - Labels',
            operations: [
                'add',
                'set',
                'remove',
            ],
            required: false,
            schema: {
                custom: 'com.atlassian.jira.plugin.system.customfieldtypes:labels',
                customId: 10071,
                items: 'string',
                type: 'array',
            },
            issueTypes: [{id: '10001', name: 'Bug'}],
        };

        const metadata = useFieldForIssueMetadata(field, 'custom1');

        let actual;
        actual = getConflictingFields([field], ['10001'], metadata);
        expect(actual.length).toBe(0);

        field = {
            ...field,
            issueTypes: [{id: '10002', name: 'Task'}],
        };

        actual = getConflictingFields([field], ['10001'], metadata);
        expect(actual.length).toBe(1);
        expect(actual[0].field.key).toEqual('custom1');
        expect(actual[0].issueTypes[0].id).toEqual('10001');
    });
});
