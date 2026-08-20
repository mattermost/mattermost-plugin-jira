// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    Instance,
    InstanceType,
    RHSErrorCode,
    RHSIssue,
} from 'types/model';
import {GlobalState, pluginStateKey} from 'types/store';
import {defaultMockState} from 'testlib/test-utils';

import {
    getConnectedCloudInstances,
    getRHSError,
    getRHSIssues,
    getRHSLoading,
    getUserConnectedInstances,
    hasCloudInstance,
    isCloudInstance,
} from './index';

function makeState(plugin: {
    installedInstances?: Instance[];
    userConnectedInstances?: Instance[];
    rhsLoading?: boolean;
    rhsIssues?: RHSIssue[];
    rhsError?: RHSErrorCode | null;
}): GlobalState {
    return {
        [pluginStateKey]: plugin,
    } as unknown as GlobalState;
}

const cloud: Instance = {instance_id: 'https://cloud.atlassian.net', type: InstanceType.CLOUD};
const cloudOAuth: Instance = {instance_id: 'https://oauth.atlassian.net', type: InstanceType.CLOUD_OAUTH};
const server: Instance = {instance_id: 'http://jira.example.com', type: InstanceType.SERVER};

describe('selectors', () => {
    test('isCloudInstance is true for CLOUD_OAUTH', () => {
        expect(isCloudInstance(cloudOAuth)).toBe(true);
    });

    test('isCloudInstance is true for CLOUD', () => {
        expect(isCloudInstance(cloud)).toBe(true);
    });

    test('isCloudInstance is false for SERVER', () => {
        expect(isCloudInstance(server)).toBe(false);
    });

    test('hasCloudInstance is true for a cloud-oauth-only installation', () => {
        const state = makeState({installedInstances: [cloudOAuth], userConnectedInstances: []});
        expect(hasCloudInstance(state)).toBe(true);
    });

    test('hasCloudInstance is true for a cloud-JWT-only installation', () => {
        const state = makeState({installedInstances: [cloud]});
        expect(hasCloudInstance(state)).toBe(true);
    });

    test('hasCloudInstance is false when only Server/DC is installed', () => {
        const state = makeState({installedInstances: [server]});
        expect(hasCloudInstance(state)).toBe(false);
    });

    test('hasCloudInstance is false when nothing is installed', () => {
        expect(hasCloudInstance(makeState({}))).toBe(false);
    });

    test('getConnectedCloudInstances returns only Cloud types', () => {
        const state = makeState({
            installedInstances: [cloud, cloudOAuth, server],
            userConnectedInstances: [cloud, cloudOAuth, server],
        });
        expect(getConnectedCloudInstances(state)).toEqual([cloud, cloudOAuth]);
    });

    test('getConnectedCloudInstances drops a connected instance that is not installed', () => {
        const state = makeState({
            installedInstances: [server],
            userConnectedInstances: [cloudOAuth],
        });
        expect(getConnectedCloudInstances(state)).toEqual([]);
    });

    test('defaultMockState yields a non-empty getUserConnectedInstances', () => {
        const state = defaultMockState as unknown as GlobalState;
        expect(getUserConnectedInstances(state).length).toBe(2);
        const connectedCloud = getConnectedCloudInstances(state);
        expect(connectedCloud).toEqual(expect.arrayContaining([
            expect.objectContaining({instance_id: 'instance1', type: InstanceType.CLOUD}),
            expect.objectContaining({instance_id: 'instance2', type: InstanceType.CLOUD_OAUTH}),
        ]));
        expect(connectedCloud).toHaveLength(2);
    });

    test('getRHSLoading and getRHSIssues keep loading distinguishable from empty', () => {
        const state = makeState({rhsLoading: true, rhsIssues: [], rhsError: null});
        expect(getRHSLoading(state)).toBe(true);
        expect(getRHSIssues(state)).toEqual([]);
    });

    test('getRHSError returns the typed code', () => {
        const state = makeState({rhsError: 'not_connected'});
        expect(getRHSError(state)).toBe('not_connected');
    });
});
