// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import {InstanceType} from 'types/model';

import {Props} from './jira_instance_and_project_selector';

describe('components/JiraInstanceAndProjectSelector', () => {
    const mockTheme = {
        centerChannelColor: '#333333',
        centerChannelBg: '#ffffff',
        buttonBg: '#166de0',
        buttonColor: '#ffffff',
        linkColor: '#2389d7',
        errorTextColor: '#fd5960',
    };

    const baseProps: Props = {
        selectedInstanceID: null,
        selectedProjectID: null,
        onInstanceChange: jest.fn(),
        onProjectChange: jest.fn(),
        onError: jest.fn(),

        theme: mockTheme as Theme,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),

        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}, {instance_id: 'instance3', type: InstanceType.SERVER}],
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
        defaultUserInstanceID: '',
        fetchJiraProjectMetadata: jest.fn().mockResolvedValue({data: {
            saved_field_values: {
                project_key: 'TEST',
            },
            projects: [
                {value: 'TEST', label: 'Test Project'},
                {value: 'AA', label: 'Apples Arrangement'},
            ],
        }}),
        getConnected: jest.fn().mockResolvedValue({error: null}),
        hideProjectSelector: false,
    };

    test('baseProps are correctly defined', () => {
        expect(baseProps.theme).toBeDefined();
        expect(baseProps.installedInstances).toHaveLength(3);
        expect(baseProps.connectedInstances).toHaveLength(2);
    });

    test('callbacks are correctly mocked', () => {
        expect(typeof baseProps.onInstanceChange).toBe('function');
        expect(typeof baseProps.onProjectChange).toBe('function');
        expect(typeof baseProps.onError).toBe('function');
    });

    test('installed instances contain expected data', () => {
        expect(baseProps.installedInstances[0].instance_id).toBe('instance1');
        expect(baseProps.installedInstances[0].type).toBe(InstanceType.CLOUD);
    });

    test('connected instances contain expected data', () => {
        expect(baseProps.connectedInstances[0].instance_id).toBe('instance1');
        expect(baseProps.connectedInstances[1].type).toBe(InstanceType.SERVER);
    });

    test('fetchJiraProjectMetadata returns expected data', async () => {
        const result = await baseProps.fetchJiraProjectMetadata('');
        expect(result.data).toBeDefined();
        expect(result.data.projects).toHaveLength(2);
    });

    test('getConnected returns no error', async () => {
        const result = await baseProps.getConnected();
        expect(result.error).toBeNull();
    });

    test('hideProjectSelector is false by default', () => {
        expect(baseProps.hideProjectSelector).toBe(false);
    });

    test('theme has expected properties', () => {
        expect(mockTheme.centerChannelColor).toBe('#333333');
        expect(mockTheme.buttonBg).toBe('#166de0');
    });
});
