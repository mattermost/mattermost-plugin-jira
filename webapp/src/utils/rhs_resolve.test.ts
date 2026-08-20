// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Instance, InstanceType, RHSTab} from 'types/model';

import {resolveRHSInstanceID, rhsTabsEqual} from './rhs_resolve';

const cloudOne: Instance = {instance_id: 'https://one.atlassian.net', type: InstanceType.CLOUD};
const cloudTwo: Instance = {instance_id: 'https://two.atlassian.net', type: InstanceType.CLOUD_OAUTH};

describe('rhs_resolve', () => {
    test('rhsTabsEqual is true for the same category key and ignores name', () => {
        const left: RHSTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};
        const right: RHSTab = {kind: 'category', name: 'Doing', key: 'indeterminate'};
        expect(rhsTabsEqual(left, right)).toBe(true);
    });

    test('rhsTabsEqual is false when status id differs', () => {
        const left: RHSTab = {kind: 'status', name: 'Blocked', id: '1'};
        const right: RHSTab = {kind: 'status', name: 'Blocked', id: '2'};
        expect(rhsTabsEqual(left, right)).toBe(false);
    });

    test('resolveRHSInstanceID prefers a saved id that is still connected', () => {
        expect(resolveRHSInstanceID(cloudTwo.instance_id, [cloudOne, cloudTwo], cloudOne.instance_id)).toBe(cloudTwo.instance_id);
    });

    test('resolveRHSInstanceID falls back to defaultUserInstanceID when saved is gone', () => {
        expect(resolveRHSInstanceID('https://gone.atlassian.net', [cloudOne, cloudTwo], cloudTwo.instance_id)).toBe(cloudTwo.instance_id);
    });

    test('resolveRHSInstanceID picks the first connected Cloud instance when nothing saved', () => {
        expect(resolveRHSInstanceID('', [cloudOne, cloudTwo], '')).toBe(cloudOne.instance_id);
    });

    test('resolveRHSInstanceID returns empty string when there are no connected Cloud instances', () => {
        expect(resolveRHSInstanceID(cloudOne.instance_id, [], cloudOne.instance_id)).toBe('');
    });
});
