// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    RHSSort,
    RHSTab,
    RHSTabKind,
    RHSViewState,
    RHS_DEFAULT_TAB,
} from 'types/model';

const storagePrefix = 'jira:rhs-view:';

export function rhsViewStorageKey(userId: string): string {
    return storagePrefix + userId;
}

function isRHSSort(value: unknown): value is RHSSort {
    return value === 'updated' || value === 'created';
}

function isRHSTabKind(value: unknown): value is RHSTabKind {
    return value === 'assigned' || value === 'category' || value === 'status';
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

    const tabRaw = candidate.tab as {kind?: unknown; name?: unknown; key?: unknown; id?: unknown};
    if (!isRHSTabKind(tabRaw.kind)) {
        return null;
    }

    const tab: RHSTab = {
        kind: tabRaw.kind,
        name: typeof tabRaw.name === 'string' && tabRaw.name ? tabRaw.name : RHS_DEFAULT_TAB.name,
    };

    if (tab.kind === 'category') {
        if (typeof tabRaw.key !== 'string' || !tabRaw.key) {
            return null;
        }
        tab.key = tabRaw.key;
    }

    if (tab.kind === 'status') {
        if (typeof tabRaw.id !== 'string' || !tabRaw.id) {
            return null;
        }
        tab.id = tabRaw.id;
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
        const tab: RHSTab = {
            kind: view.tab.kind,
            name: view.tab.name,
        };
        if (view.tab.key) {
            tab.key = view.tab.key;
        }
        if (view.tab.id) {
            tab.id = view.tab.id;
        }
        const payload: RHSViewState = {
            instance: view.instance,
            tab,
            sort: view.sort,
        };
        localStorage.setItem(rhsViewStorageKey(userId), JSON.stringify(payload));
    } catch {
        // ignore quota / private-mode
    }
}
