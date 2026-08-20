// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Instance, RHSTab} from 'types/model';

export function rhsTabsEqual(a: RHSTab, b: RHSTab): boolean {
    return a.kind === b.kind &&
        (a.key || '') === (b.key || '') &&
        (a.id || '') === (b.id || '');
}

export function resolveRHSInstanceID(
    savedInstanceID: string,
    connectedCloud: Instance[],
    defaultUserInstanceID: string,
): string {
    if (savedInstanceID && connectedCloud.some((instance) => instance.instance_id === savedInstanceID)) {
        return savedInstanceID;
    }
    if (defaultUserInstanceID && connectedCloud.some((instance) => instance.instance_id === defaultUserInstanceID)) {
        return defaultUserInstanceID;
    }
    if (connectedCloud.length >= 1) {
        return connectedCloud[0].instance_id;
    }
    return '';
}
