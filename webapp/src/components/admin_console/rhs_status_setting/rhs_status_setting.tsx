// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useRef, useState} from 'react';
import {MultiValueRemoveProps, components} from 'react-select';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import ReactSelectSetting from 'components/react_select_setting';
import {
    Instance,
    RHSErrorCode,
    RHSStatusesResponse,
    ReactSelectOption,
} from 'types/model';

import {RHSFetchError} from '../../../client';

import {
    ASSIGNED_OPTION_VALUE,
    ASSIGNED_TAB,
    INSTANCE_LABEL,
    NO_CLOUD_INSTANCE_MESSAGE,
    RHSStatusTabsValue,
    STATUS_TABS_HELP,
    STATUS_TABS_LABEL,
    StatusOptionGroup,
    StatusTabOption,
    UNABLE_TO_LOAD_STATUSES_MESSAGE,
    buildPersistedValue,
    buildStatusOptionGroups,
    displayTabsForInstance,
    extrasFromOptionValues,
    filterInstalledCloudInstances,
    flattenOptionGroups,
    isTabsValueEmpty,
    isVirtualSeedExtras,
    optionsFromTabs,
    persistedValuesEqual,
    statusFetchMessage,
    storedExtrasForInstance,
} from './rhs_status_options';

export type Props = {
    id: string;
    value: RHSStatusTabsValue | null;
    disabled: boolean;
    config: unknown;
    onChange: (id: string, value: RHSStatusTabsValue) => void;
    setSaveNeeded: () => void;
    theme: Theme;
    installedInstances: Instance[];
    fetchRHSStatuses: (instanceID: string) => Promise<{data?: RHSStatusesResponse; error?: RHSFetchError}>;
    getConnected: () => Promise<{data?: unknown; error?: unknown}>;
};

type ConsoleSelectProps = {
    name: string;
    inputId: string;
    label: string;
    helpText?: string;
    options: unknown;
    value: unknown;
    isMulti: boolean;
    isClearable: boolean;
    isDisabled: boolean;
    isLoading?: boolean;
    theme: Theme;
    components?: unknown;
    onChange: (name: string, value: string | string[] | null) => void;
};

const ConsoleSelect = ReactSelectSetting as unknown as React.ComponentType<ConsoleSelectProps>;

function RHSFixedMultiValueRemove(props: MultiValueRemoveProps<ReactSelectOption>): React.ReactElement | null {
    const option = props.data as StatusTabOption;
    if (option.isFixed) {
        return null;
    }

    return (
        <components.MultiValueRemove
            {...props}
        />
    );
}

export default function RHSStatusSetting(props: Props): React.ReactElement {
    const {
        id,
        value,
        disabled,
        onChange,
        setSaveNeeded,
        theme,
        installedInstances,
        fetchRHSStatuses,
        getConnected,
    } = props;

    const [instanceID, setInstanceID] = useState('');
    const [optionGroups, setOptionGroups] = useState<StatusOptionGroup[]>([]);
    const [fetchError, setFetchError] = useState<RHSErrorCode | null>(null);
    const [loadingStatuses, setLoadingStatuses] = useState(false);
    const latestGeneration = useRef(0);

    const cloudInstances = filterInstalledCloudInstances(installedInstances);
    const selectedTabs = instanceID ? displayTabsForInstance(value, instanceID) : [ASSIGNED_TAB];
    const selectedOptions = optionsFromTabs(selectedTabs);
    const flatOptions = flattenOptionGroups(optionGroups);

    useEffect(() => {
        const refresh = async (): Promise<void> => {
            try {
                await getConnected();
            } catch {
                // best-effort; must not call onChange
            }
        };

        refresh();
    }, [getConnected]);

    useEffect(() => {
        if (instanceID) {
            return;
        }

        const first = filterInstalledCloudInstances(installedInstances)[0];
        if (!first) {
            return;
        }

        setInstanceID(first.instance_id);
    }, [instanceID, installedInstances]);

    useEffect(() => {
        if (!instanceID) {
            return;
        }

        latestGeneration.current += 1;
        const generation = latestGeneration.current;
        setLoadingStatuses(true);
        setFetchError(null);

        const load = async (): Promise<void> => {
            let result: {data?: RHSStatusesResponse; error?: RHSFetchError} = {};
            try {
                result = await fetchRHSStatuses(instanceID);
            } catch {
                result = {error: new RHSFetchError('internal_error', UNABLE_TO_LOAD_STATUSES_MESSAGE, 0)};
            }

            if (generation !== latestGeneration.current) {
                return;
            }

            if (result.error) {
                setFetchError(result.error.errorCode);
                setOptionGroups([]);
            } else if (result.data) {
                setOptionGroups(buildStatusOptionGroups(result.data));
                setFetchError(null);
            }
            setLoadingStatuses(false);
        };

        load();
    }, [instanceID, fetchRHSStatuses]);

    const handleInstanceChange = (name: string, nextID: string | string[] | null): void => {
        if (!nextID || Array.isArray(nextID) || nextID === instanceID) {
            return;
        }

        setInstanceID(nextID);
    };

    const handleStatusValuesChange = (name: string, values: string | string[] | null): void => {
        if (disabled || fetchError !== null || !instanceID || !Array.isArray(values)) {
            return;
        }

        const nextValues = values.slice();
        if (nextValues.indexOf(ASSIGNED_OPTION_VALUE) === -1) {
            nextValues.unshift(ASSIGNED_OPTION_VALUE);
        }

        const extras = extrasFromOptionValues(nextValues, [...flatOptions, ...selectedOptions]);
        const emptyValue = isTabsValueEmpty(value, instanceID);
        if (emptyValue && isVirtualSeedExtras(extras)) {
            return;
        }
        if (!emptyValue && JSON.stringify(storedExtrasForInstance(value, instanceID)) === JSON.stringify(extras)) {
            return;
        }

        const next = buildPersistedValue(value, instanceID, extras);
        if (persistedValuesEqual(value, next)) {
            return;
        }

        onChange(id, next);
        setSaveNeeded();
    };

    if (cloudInstances.length === 0) {
        return (
            <p>
                {NO_CLOUD_INSTANCE_MESSAGE}
            </p>
        );
    }

    const instanceOptions = cloudInstances.map((instance) => {
        return {
            label: instance.alias || instance.instance_id,
            value: instance.instance_id,
        };
    });
    const selectedInstance = instanceOptions.find((option) => option.value === instanceID) || null;
    const statusHelp = fetchError === null ? STATUS_TABS_HELP : statusFetchMessage(fetchError);

    return (
        <React.Fragment>
            <ConsoleSelect
                name={'rhs-status-instance'}
                inputId={'rhs-status-instance'}
                label={INSTANCE_LABEL}
                options={instanceOptions}
                value={selectedInstance}
                isMulti={false}
                isClearable={false}
                isDisabled={disabled}
                theme={theme}
                onChange={handleInstanceChange}
            />
            <ConsoleSelect
                name={'rhs-status-tabs'}
                inputId={'rhs-status-tabs'}
                label={STATUS_TABS_LABEL}
                helpText={statusHelp}
                isMulti={true}
                isClearable={false}
                isLoading={loadingStatuses}
                isDisabled={disabled || !instanceID || fetchError !== null}
                options={optionGroups}
                value={selectedOptions}
                theme={theme}
                components={{MultiValueRemove: RHSFixedMultiValueRemove}}
                onChange={handleStatusValuesChange}
            />
        </React.Fragment>
    );
}
