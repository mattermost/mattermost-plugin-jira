// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {Combobox} from '@mattermost/compass-ui/components/combobox';

import Setting, {REQUIRED_MESSAGE, fieldLabel} from 'components/setting';

import {Z_INDEX_MENU_PORTAL} from 'utils/styles';
import uniqueId from 'utils/unique_id';

import {SelectOption} from 'types/model';

const MAX_NUM_OPTIONS = 100;

export type SelectValue = SelectOption | SelectOption[] | null | undefined;

export type Props = {
    name?: string;
    label?: React.ReactNode;
    'aria-label'?: string;
    options?: SelectOption[] | null;
    value?: SelectValue;
    onChange?(name: string | undefined, value: any): void;
    isMulti?: boolean;
    required?: boolean;
    hideRequiredStar?: boolean;
    isDisabled?: boolean;
    isLoading?: boolean;
    isClearable?: boolean;
    placeholder?: string;
    addValidate?: (isValid: () => boolean) => void;
    removeValidate?: (isValid: () => boolean) => void;
    allowUserDefinedValue?: boolean;
    limitOptions?: boolean;
    resetInvalidOnChange?: boolean;
};

type State = {
    invalid: boolean;
    query: string;
};

const toArray = (value: SelectValue): SelectOption[] => {
    if (!value) {
        return [];
    }
    return Array.isArray(value) ? value : [value];
};

const valueKey = (value: SelectValue) => toArray(value).map((opt) => opt.value).join(',');

export default class SelectSetting extends React.PureComponent<Props, State> {
    // Unique per instance so repeated selects (one per filter row) never share an id.
    private fallbackId = uniqueId('jira-select-setting');
    private fieldId = this.props.name ? uniqueId(this.props.name) : this.fallbackId;

    state: State = {invalid: false, query: ''};

    componentDidMount() {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }
    }

    componentWillUnmount() {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    componentDidUpdate(prevProps: Props) {
        if (this.state.invalid && valueKey(this.props.value) !== valueKey(prevProps.value)) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleChange = (value: string | string[] | null) => {
        if (this.props.onChange) {
            if (this.props.isMulti) {
                this.props.onChange(this.props.name, Array.isArray(value) ? value : []);
            } else {
                this.props.onChange(this.props.name, value || null);
            }
        }

        const nextState: Partial<State> = {query: ''};
        if (this.props.resetInvalidOnChange) {
            nextState.invalid = false;
        }
        this.setState(nextState as State);
    };

    handleInputChange = (query: string) => {
        this.setState({query});
    };

    handleBlur = () => {
        this.setState({query: ''});
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        const valid = toArray(this.props.value).length > 0;
        this.setState({invalid: !valid});
        return valid;
    };

    isLimited = () => {
        return Boolean(this.props.limitOptions && this.props.options && this.props.options.length > MAX_NUM_OPTIONS);
    };

    getVisibleOptions = (): SelectOption[] => {
        const options = this.props.options || [];
        if (!this.isLimited()) {
            return options;
        }

        const query = this.state.query.toUpperCase();
        const filtered = query ? options.filter((opt) => opt.label.toUpperCase().includes(query)) : options;
        return filtered.slice(0, MAX_NUM_OPTIONS);
    };

    render() {
        const {label, required, hideRequiredStar, isMulti} = this.props;
        const selected = toArray(this.props.value);
        const values = selected.map((opt) => opt.value);
        const limited = this.isLimited();
        const ariaLabel = this.props['aria-label'];

        const error = required && this.state.invalid ? REQUIRED_MESSAGE : null;
        let errorId;
        if (error) {
            errorId = `${this.fieldId}-error`;
        }

        return (
            <Setting
                error={error}
                errorId={errorId}
            >
                <Combobox
                    id={this.fieldId}
                    aria-describedby={errorId}
                    label={label && fieldLabel(label, required && !hideRequiredStar)}
                    aria-label={ariaLabel}
                    listboxLabel={typeof label === 'string' ? label : ariaLabel}
                    options={this.getVisibleOptions()}
                    selectedOptions={selected}
                    multiple={isMulti}
                    value={isMulti ? values : (values[0] ?? null)}
                    onChange={this.handleChange}
                    filter={!limited}
                    onInputChange={this.handleInputChange}
                    onBlur={this.handleBlur}
                    creatable={this.props.allowUserDefinedValue}
                    formatCreateLabel={(value) => `Add "${value}"`}
                    {...(this.props.allowUserDefinedValue && {emptyMessage: 'Start typing...'})}
                    clearable={this.props.isClearable}
                    disabled={this.props.isDisabled}
                    loading={this.props.isLoading}
                    invalid={required && this.state.invalid}
                    required={required}
                    placeholder={this.props.placeholder}
                    submitOnEnter={false}
                    zIndex={Z_INDEX_MENU_PORTAL}
                />
            </Setting>
        );
    }
}
