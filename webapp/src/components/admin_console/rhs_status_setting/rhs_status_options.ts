// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Instance, isCloudInstance} from 'types/model';
import {
    RHSErrorCode,
    RHSStatus,
    RHSStatusesResponse,
    RHSTab,
    RHS_DEFAULT_TAB,
    rhsTabIdentity,
} from 'types/rhs';

export type RHSStatusTabsValue = Record<string, RHSTab[]>;

export type StatusTabOption = {
    label: string;
    value: string;
    isFixed?: boolean;
    tab: RHSTab;
    description?: string;
};

export const ASSIGNED_TAB: RHSTab = RHS_DEFAULT_TAB;

export const IN_PROGRESS_TAB: RHSTab = {
    kind: 'category',
    key: 'indeterminate',
    name: 'In Progress',
};

export const ASSIGNED_OPTION_VALUE = rhsTabIdentity(ASSIGNED_TAB);

export const ASSIGNED_OPTION: StatusTabOption = {
    label: ASSIGNED_TAB.name,
    value: ASSIGNED_OPTION_VALUE,
    isFixed: true,
    tab: ASSIGNED_TAB,
};

export const CATEGORY_OPTION_PREFIX = 'category:';
export const STATUS_OPTION_PREFIX = 'status:';

export const NOT_CONNECTED_MESSAGE = 'Connect Jira to change status tabs.';
export const STATUS_TABS_HELP = 'Choosing a category name (To Do, In Progress, Done) matches statusCategory and is broader than a single status.';
export const INSTANCE_LABEL = 'Instance';
export const STATUS_TABS_LABEL = 'Status tabs';
export const NO_CLOUD_INSTANCE_MESSAGE = 'Install a Jira Cloud instance to configure RHS status tabs.';
export const UNABLE_TO_LOAD_STATUSES_MESSAGE = 'Unable to load statuses.';

export function filterInstalledCloudInstances(instances: Instance[] | null): Instance[] {
    if (!instances) {
        return [];
    }
    return instances.filter(isCloudInstance);
}

export function isTabsValueUnset(value: RHSStatusTabsValue | null, instanceID: string): boolean {
    if (!value || typeof value !== 'object') {
        return true;
    }
    return value[instanceID] == null;
}

export function storedExtrasForInstance(value: RHSStatusTabsValue | null, instanceID: string): RHSTab[] {
    if (!value || isTabsValueUnset(value, instanceID)) {
        return [];
    }
    return value[instanceID].filter((tab) => tab.kind !== 'assigned');
}

export function displayTabsForInstance(value: RHSStatusTabsValue | null, instanceID: string): RHSTab[] {
    if (isTabsValueUnset(value, instanceID)) {
        return [ASSIGNED_TAB, IN_PROGRESS_TAB];
    }
    return [ASSIGNED_TAB, ...storedExtrasForInstance(value, instanceID)];
}

export function tabFromOptionValue(value: string, options: StatusTabOption[]): RHSTab | null {
    const match = options.find((option) => option.value === value);
    if (match) {
        return match.tab;
    }
    if (value === ASSIGNED_OPTION_VALUE) {
        return ASSIGNED_TAB;
    }
    if (value.indexOf(CATEGORY_OPTION_PREFIX) === 0) {
        const key = value.slice(CATEGORY_OPTION_PREFIX.length);
        if (!key) {
            return null;
        }
        return {
            kind: 'category',
            key,
            name: key,
        };
    }
    if (value.indexOf(STATUS_OPTION_PREFIX) === 0) {
        const id = value.slice(STATUS_OPTION_PREFIX.length);
        if (!id) {
            return null;
        }
        return {
            kind: 'status',
            id,
            name: id,
        };
    }
    return null;
}

export function optionsFromTabs(tabs: RHSTab[]): StatusTabOption[] {
    return tabs.map((tab) => {
        if (tab.kind === 'assigned') {
            return ASSIGNED_OPTION;
        }
        return {
            label: tab.name,
            value: rhsTabIdentity(tab),
            tab,
        };
    });
}

export function extrasFromOptionValues(values: string[], options: StatusTabOption[]): RHSTab[] {
    const extras: RHSTab[] = [];
    for (let i = 0; i < values.length; i++) {
        const value = values[i];
        if (value === ASSIGNED_OPTION_VALUE) {
            continue;
        }
        const tab = tabFromOptionValue(value, options);
        if (tab && tab.kind !== 'assigned') {
            extras.push(tab);
        }
    }
    return extras;
}

export function buildPersistedValue(
    current: RHSStatusTabsValue | null,
    instanceID: string,
    extras: RHSTab[],
): RHSStatusTabsValue {
    const next: RHSStatusTabsValue = current && typeof current === 'object' ? {...current} : {};
    next[instanceID] = extras;
    return next;
}

export function persistedValuesEqual(left: RHSStatusTabsValue | null, right: RHSStatusTabsValue): boolean {
    return JSON.stringify(left || {}) === JSON.stringify(right);
}

export function dedupeStatusesById(statuses: RHSStatus[]): RHSStatus[] {
    const keys = new Set<string>();
    const out: RHSStatus[] = [];
    for (let i = 0; i < statuses.length; i++) {
        const status = statuses[i];
        if (!keys.has(status.id)) {
            keys.add(status.id);
            out.push(status);
        }
    }
    return out;
}

export type StatusOptionGroup = {
    label: string;
    options: StatusTabOption[];
};

export function buildStatusOptionGroups(data: RHSStatusesResponse): StatusOptionGroup[] {
    const categories = data.categories
        .filter((category) => category.key !== 'undefined')
        .map((category): StatusTabOption => {
            const tab: RHSTab = {
                kind: 'category',
                key: category.key,
                name: category.name,
            };
            return {
                label: category.name,
                value: rhsTabIdentity(tab),
                tab,
            };
        });

    const statuses = dedupeStatusesById(data.statuses).map((status): StatusTabOption => {
        const tab: RHSTab = {
            kind: 'status',
            id: status.id,
            name: status.name,
        };
        const option: StatusTabOption = {
            label: status.name,
            value: rhsTabIdentity(tab),
            tab,
        };
        const description = statusProjectDescription(status);
        if (description) {
            option.description = description;
        }
        return option;
    });

    return [
        {label: 'Categories', options: categories},
        {label: 'Statuses', options: statuses},
    ];
}

export function flattenOptionGroups(groups: StatusOptionGroup[]): StatusTabOption[] {
    const out: StatusTabOption[] = [ASSIGNED_OPTION];
    for (let i = 0; i < groups.length; i++) {
        out.push(...groups[i].options);
    }
    return out;
}

export function statusProjectDescription(status: RHSStatus): string {
    if (!status.project) {
        return '';
    }
    const name = status.project.name || '';
    const key = status.project.key || '';
    if (name && key) {
        return name + ' (' + key + ')';
    }
    if (name) {
        return name;
    }
    if (key) {
        return key;
    }
    return '';
}

export function statusTabOptionMatches(option: StatusTabOption, input: string): boolean {
    const query = input.trim().toLowerCase();
    if (!query) {
        return true;
    }
    if (option.label.toLowerCase().indexOf(query) !== -1) {
        return true;
    }
    return Boolean(option.description && option.description.toLowerCase().indexOf(query) !== -1);
}

export function statusFetchMessage(code: RHSErrorCode): string {
    switch (code) {
    case 'not_connected':
        return NOT_CONNECTED_MESSAGE;
    case 'rate_limited':
        return 'Jira is rate limiting requests, try again shortly';
    case 'not_authorized':
        return 'not authorized';
    case 'not_cloud':
        return 'Jira RHS is available for Jira Cloud only';
    case 'invalid_request':
    case 'internal_error':
        return UNABLE_TO_LOAD_STATUSES_MESSAGE;
    default: {
        const exhaustive: never = code;
        return exhaustive;
    }
    }
}
