// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {RHSSort, RHSViewState, rhsTabFromAPI} from 'types/rhs';

const storagePrefix = 'jira:rhs-view:';

export function rhsViewStorageKey(userId: string): string {
    return storagePrefix + userId;
}

function isRHSSort(value: unknown): value is RHSSort {
    return value === 'updated' || value === 'created';
}

export function validateRHSViewState(raw: unknown): RHSViewState | null {
    if (!raw || typeof raw !== 'object') {
        return null;
    }

    const candidate = raw as {instance?: unknown; tab?: unknown; sort?: unknown};
    if (typeof candidate.instance !== 'string' || !candidate.instance) {
        return null;
    }
    if (!isRHSSort(candidate.sort)) {
        return null;
    }
    if (!candidate.tab || typeof candidate.tab !== 'object') {
        return null;
    }

    const tab = rhsTabFromAPI(candidate.tab as {kind?: unknown; name?: unknown; key?: unknown; id?: unknown});
    if (!tab) {
        return null;
    }

    return {
        instance: candidate.instance,
        tab,
        sort: candidate.sort,
    };
}

export function loadRHSViewState(userId: string): RHSViewState | null {
    if (!userId) {
        return null;
    }

    try {
        const raw = localStorage.getItem(rhsViewStorageKey(userId));
        if (!raw) {
            return null;
        }
        return validateRHSViewState(JSON.parse(raw));
    } catch {
        return null;
    }
}

export function saveRHSViewState(userId: string, view: RHSViewState): void {
    if (!userId) {
        return;
    }

    try {
        const payload: RHSViewState = {
            instance: view.instance,
            tab: view.tab,
            sort: view.sort,
        };
        localStorage.setItem(rhsViewStorageKey(userId), JSON.stringify(payload));
    } catch {
        // ignore quota / private-mode
    }
}
