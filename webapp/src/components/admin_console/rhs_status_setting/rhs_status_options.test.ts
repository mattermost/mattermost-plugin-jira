// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {InstanceType} from 'types/model';
import {RHSStatusesResponse} from 'types/rhs';

import {
    ASSIGNED_TAB,
    IN_PROGRESS_TAB,
    NOT_CONNECTED_MESSAGE,
    buildPersistedValue,
    buildStatusOptionGroups,
    displayTabsForInstance,
    filterInstalledCloudInstances,
    flattenOptionGroups,
    isTabsValueUnset,
    statusFetchMessage,
    statusProjectDescription,
    statusTabOptionMatches,
} from './rhs_status_options';

const CLOUD_ID = 'https://cloud.example.atlassian.net';
const OAUTH_ID = 'https://oauth.example.atlassian.net';
const SERVER_ID = 'http://jira.example.com';

const statusesData: RHSStatusesResponse = {
    categories: [
        {id: 1, key: 'undefined', name: 'No Category'},
        {id: 2, key: 'new', name: 'To Do'},
        {id: 4, key: 'indeterminate', name: 'In Progress'},
        {id: 3, key: 'done', name: 'Done'},
    ],
    statuses: [
        {id: '3', name: 'In Progress', statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'}},
        {id: '3', name: 'In Progress', statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'}},
        {id: '10001', name: 'Submitted', statusCategory: {id: 2, key: 'new', name: 'To Do'}},
        {
            id: '10042',
            name: 'In Progress',
            statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'},
            project: {id: '10000', key: 'PLAY', name: 'Playbooks'},
        },
    ],
};

describe('rhs_status_options', () => {
    test('isTabsValueUnset treats null undefined empty object and missing key as unset', () => {
        const missing: {value?: Record<string, never> | null} = {};

        expect(isTabsValueUnset(null, CLOUD_ID)).toBe(true);
        expect(isTabsValueUnset(missing.value as never, CLOUD_ID)).toBe(true);
        expect(isTabsValueUnset({}, CLOUD_ID)).toBe(true);
        expect(isTabsValueUnset({[OAUTH_ID]: [IN_PROGRESS_TAB]}, CLOUD_ID)).toBe(true);
        expect(isTabsValueUnset({[CLOUD_ID]: []}, CLOUD_ID)).toBe(false);
    });

    test('displayTabsForInstance returns Assigned and In Progress when value is unset', () => {
        expect(displayTabsForInstance(null, CLOUD_ID)).toEqual([ASSIGNED_TAB, IN_PROGRESS_TAB]);
        expect(displayTabsForInstance({}, CLOUD_ID)).toEqual([ASSIGNED_TAB, IN_PROGRESS_TAB]);
    });

    test('displayTabsForInstance returns Assigned only for an explicit empty extras list', () => {
        expect(displayTabsForInstance({[CLOUD_ID]: []}, CLOUD_ID)).toEqual([ASSIGNED_TAB]);
    });

    test('displayTabsForInstance prepends Assigned to stored extras and drops a stored assigned entry', () => {
        const doneTab = {kind: 'category' as const, key: 'done', name: 'Done'};
        const value = {
            [CLOUD_ID]: [
                {kind: 'assigned' as const, name: 'Assigned'},
                doneTab,
            ],
        };

        expect(displayTabsForInstance(value, CLOUD_ID)).toEqual([ASSIGNED_TAB, doneTab]);
    });

    test('buildPersistedValue writes In Progress extras on an unset instance', () => {
        expect(buildPersistedValue(null, CLOUD_ID, [IN_PROGRESS_TAB])).toEqual({
            [CLOUD_ID]: [IN_PROGRESS_TAB],
        });
    });

    test('buildPersistedValue writes an empty extras list for Assigned only', () => {
        expect(buildPersistedValue(null, CLOUD_ID, [])).toEqual({
            [CLOUD_ID]: [],
        });
    });

    test('buildPersistedValue writes extras without Assigned', () => {
        const submitted = {kind: 'status' as const, id: '10001', name: 'Submitted'};
        const extras = [IN_PROGRESS_TAB, submitted];

        expect(buildPersistedValue(null, CLOUD_ID, extras)).toEqual({
            [CLOUD_ID]: extras,
        });
        expect(buildPersistedValue(null, CLOUD_ID, extras)[CLOUD_ID].some((tab) => tab.kind === 'assigned')).toBe(false);
    });

    test('buildStatusOptionGroups omits No Category and dedupes statuses by id', () => {
        const groups = buildStatusOptionGroups(statusesData);
        const options = flattenOptionGroups(groups);

        expect(options.some((option) => option.tab.kind === 'category' && option.tab.key === 'undefined')).toBe(false);
        expect(options.filter((option) => option.value === 'status:3')).toHaveLength(1);
    });

    test('buildStatusOptionGroups puts the project on status option descriptions', () => {
        const groups = buildStatusOptionGroups(statusesData);
        const statuses = groups.find((group) => group.label === 'Statuses');
        const playbooks = statuses?.options.find((option) => option.value === 'status:10042');
        const globalInProgress = statuses?.options.find((option) => option.value === 'status:3');

        expect(playbooks?.description).toBe('Playbooks (PLAY)');
        expect(globalInProgress?.description).toBeUndefined();
    });

    test('statusProjectDescription prefers name and key together', () => {
        expect(statusProjectDescription({
            id: '1',
            name: 'In Progress',
            statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'},
            project: {id: '10000', key: 'PLAY', name: 'Playbooks'},
        })).toBe('Playbooks (PLAY)');
        expect(statusProjectDescription({
            id: '1',
            name: 'In Progress',
            statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'},
            project: {name: 'Playbooks'},
        })).toBe('Playbooks');
        expect(statusProjectDescription({
            id: '1',
            name: 'In Progress',
            statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'},
            project: {key: 'PLAY'},
        })).toBe('PLAY');
        expect(statusProjectDescription({
            id: '1',
            name: 'In Progress',
            statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'},
        })).toBe('');
    });

    test('statusTabOptionMatches searches label and project description', () => {
        const option = {
            label: 'In Progress',
            value: 'status:10042',
            description: 'Playbooks (PLAY)',
            tab: {kind: 'status' as const, id: '10042', name: 'In Progress'},
        };

        expect(statusTabOptionMatches(option, '')).toBe(true);
        expect(statusTabOptionMatches(option, 'in')).toBe(true);
        expect(statusTabOptionMatches(option, 'playbooks')).toBe(true);
        expect(statusTabOptionMatches(option, 'PLAY')).toBe(true);
        expect(statusTabOptionMatches(option, 'Design')).toBe(false);
    });

    test('filterInstalledCloudInstances drops SERVER and keeps cloud and cloud-oauth', () => {
        const filtered = filterInstalledCloudInstances([
            {instance_id: CLOUD_ID, type: InstanceType.CLOUD},
            {instance_id: OAUTH_ID, type: InstanceType.CLOUD_OAUTH},
            {instance_id: SERVER_ID, type: InstanceType.SERVER},
        ]);

        expect(filtered.map((instance) => instance.instance_id)).toEqual([CLOUD_ID, OAUTH_ID]);
    });

    test('statusFetchMessage returns the connect copy for not_connected', () => {
        expect(statusFetchMessage('not_connected')).toBe(NOT_CONNECTED_MESSAGE);
    });
});
