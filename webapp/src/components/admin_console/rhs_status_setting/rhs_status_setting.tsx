// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useRef, useState} from 'react';
import ReactSelect, {
    FilterOptionOption,
    FormatOptionLabelMeta,
    GroupBase,
    MultiValue,
    MultiValueProps,
    MultiValueRemoveProps,
    StylesConfig,
    components,
} from 'react-select';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import Setting from 'components/setting';
import {Instance, ReactSelectOption} from 'types/model';
import {RHSErrorCode} from 'types/rhs';
import {getStyleForReactSelect} from 'utils/styles';

import {RHSFetchError, getRHSStatuses} from '../../../client/rhs';

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
    buildPersistedValue,
    buildStatusOptionGroups,
    displayTabsForInstance,
    extrasFromOptionValues,
    filterInstalledCloudInstances,
    flattenOptionGroups,
    isTabsValueUnset,
    optionsFromTabs,
    persistedValuesEqual,
    statusFetchMessage,
    statusTabOptionMatches,
    storedExtrasForInstance,
} from './rhs_status_options';

export type Props = {
    id: string;
    value: RHSStatusTabsValue | null;
    disabled: boolean;
    onChange: (id: string, value: RHSStatusTabsValue) => void;
    setSaveNeeded: () => void;
    theme: Theme;
    installedInstances: Instance[];
    pluginServerRoute: string;
    getConnected: () => Promise<{data?: unknown; error?: unknown}>;
};

function RHSFixedMultiValue(props: MultiValueProps<StatusTabOption, true>): React.ReactElement {
    if (!props.data.isFixed) {
        return (
            <components.MultiValue
                {...props}
            />
        );
    }

    return (
        <components.MultiValue
            {...props}
            innerProps={{
                ...props.innerProps,
                style: {
                    ...props.innerProps?.style,
                    paddingRight: 8,
                },
            }}
        />
    );
}

function RHSFixedMultiValueRemove(props: MultiValueRemoveProps<StatusTabOption, true>): React.ReactElement | null {
    if (props.data.isFixed) {
        return null;
    }

    return (
        <components.MultiValueRemove
            {...props}
        />
    );
}

function formatStatusTabOption(data: StatusTabOption, meta: FormatOptionLabelMeta<StatusTabOption>): React.ReactNode {
    if (meta.context === 'value' || !data.description) {
        return data.label;
    }

    return (
        <div>
            <div>{data.label}</div>
            <div style={{opacity: 0.6}}>
                {data.description}
            </div>
        </div>
    );
}

function filterStatusTabOption(option: FilterOptionOption<StatusTabOption>, input: string): boolean {
    return statusTabOptionMatches(option.data, input);
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
        pluginServerRoute,
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
            try {
                const data = await getRHSStatuses(pluginServerRoute, instanceID);
                if (generation !== latestGeneration.current) {
                    return;
                }
                setOptionGroups(buildStatusOptionGroups(data));
                setFetchError(null);
            } catch (error) {
                if (generation !== latestGeneration.current) {
                    return;
                }
                const code = error instanceof RHSFetchError ? error.errorCode : 'internal_error';
                setFetchError(code);
                setOptionGroups([]);
            }
            if (generation === latestGeneration.current) {
                setLoadingStatuses(false);
            }
        };

        load();
    }, [instanceID, pluginServerRoute]);

    const handleInstanceChange = (next: ReactSelectOption | null): void => {
        if (!next || next.value === instanceID) {
            return;
        }

        setInstanceID(next.value);
    };

    const persistExtrasFromValues = (values: string[]): void => {
        if (disabled || fetchError !== null || !instanceID) {
            return;
        }

        const nextValues = values.slice();
        if (nextValues.indexOf(ASSIGNED_OPTION_VALUE) === -1) {
            nextValues.unshift(ASSIGNED_OPTION_VALUE);
        }

        const extras = extrasFromOptionValues(nextValues, [...flatOptions, ...selectedOptions]);
        if (!isTabsValueUnset(value, instanceID) && JSON.stringify(storedExtrasForInstance(value, instanceID)) === JSON.stringify(extras)) {
            return;
        }

        const next = buildPersistedValue(value, instanceID, extras);
        if (persistedValuesEqual(value, next)) {
            return;
        }

        onChange(id, next);
        setSaveNeeded();
    };

    const handleStatusSelectChange = (next: MultiValue<StatusTabOption>): void => {
        persistExtrasFromValues(next.map((option) => option.value));
    };

    if (cloudInstances.length === 0) {
        return (
            <p>
                {NO_CLOUD_INSTANCE_MESSAGE}
            </p>
        );
    }

    const instanceOptions: ReactSelectOption[] = cloudInstances.map((instance) => {
        return {
            label: instance.alias || instance.instance_id,
            value: instance.instance_id,
        };
    });
    const selectedInstance = instanceOptions.find((option) => option.value === instanceID) || null;
    const statusHelp = fetchError === null ? STATUS_TABS_HELP : statusFetchMessage(fetchError);

    return (
        <React.Fragment>
            <Setting
                inputId={'rhs-status-instance'}
                label={INSTANCE_LABEL}
            >
                <ReactSelect<ReactSelectOption, false>
                    inputId={'rhs-status-instance'}
                    options={instanceOptions}
                    value={selectedInstance}
                    isClearable={false}
                    isDisabled={disabled}
                    onChange={handleInstanceChange}
                    styles={getStyleForReactSelect(theme) as StylesConfig<ReactSelectOption, false>}
                    menuPortalTarget={document.body}
                    menuPlacement={'auto'}
                    aria-label={INSTANCE_LABEL}
                />
            </Setting>
            <Setting
                inputId={'rhs-status-tabs'}
                label={STATUS_TABS_LABEL}
                helpText={statusHelp}
            >
                <ReactSelect<StatusTabOption, true, GroupBase<StatusTabOption>>
                    inputId={'rhs-status-tabs'}
                    isMulti={true}
                    isClearable={false}
                    isLoading={loadingStatuses}
                    isDisabled={disabled || !instanceID || fetchError !== null}
                    options={optionGroups}
                    value={selectedOptions}
                    components={{MultiValue: RHSFixedMultiValue, MultiValueRemove: RHSFixedMultiValueRemove}}
                    formatOptionLabel={formatStatusTabOption}
                    filterOption={filterStatusTabOption}
                    onChange={handleStatusSelectChange}
                    styles={getStyleForReactSelect(theme) as StylesConfig<StatusTabOption, true>}
                    menuPortalTarget={document.body}
                    menuPlacement={'auto'}
                    aria-label={STATUS_TABS_LABEL}
                />
            </Setting>
        </React.Fragment>
    );
}
