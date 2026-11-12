// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import debounce from 'debounce-promise';

import {Combobox} from '@mattermost/compass-ui/components/combobox';

import {IssueMetadata, SelectOption} from 'types/model';

import {Z_INDEX_MENU_PORTAL} from 'utils/styles';
import uniqueId from 'utils/unique_id';

import {Props as SelectSettingProps} from 'components/select_setting';
import Setting, {REQUIRED_MESSAGE, fieldLabel} from 'components/setting';
import {TEAM_FIELD} from '../../constant';

const searchDebounceDelay = 400;

export type Props = Pick<SelectSettingProps,
'label' | 'aria-label' | 'required' | 'hideRequiredStar' | 'isDisabled' | 'isClearable' |
'placeholder' | 'resetInvalidOnChange' | 'addValidate' | 'removeValidate'
> & {
    name?: string;
    onChange(values: string | string[]): void;
    value?: string | string[];
    fetchInitialSelectedValues: () => Promise<SelectOption[]>;
    search: (searchTerm: string) => Promise<SelectOption[]>;
    isMulti?: boolean;
    issueMetadata?: IssueMetadata;
    instanceID: string;
    fieldKey?: string;
};

type State = {
    invalid: boolean;
    error?: string;
    loading: boolean;
    options: SelectOption[];
    defaultOptions: SelectOption[] | null;
    cachedSelectedOptions: SelectOption[];
};

const toValues = (value: Props['value'] | null): string[] => {
    if (!value) {
        return [];
    }
    return (Array.isArray(value) ? value : [value]).map(String);
};

export default class BackendSelector extends React.PureComponent<Props, State> {
    private fallbackId = uniqueId('jira-backend-selector');

    static defaultProps = {
        isMulti: false,
    };

    private latestSearch = 0;

    state: State = {
        invalid: false,
        error: '',
        loading: false,
        options: [],
        defaultOptions: null,
        cachedSelectedOptions: [],
    };

    componentDidMount(): void {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }

        this.loadInitialOptions();
    }

    componentWillUnmount(): void {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    componentDidUpdate(prevProps: Props): void {
        if (this.state.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    private loadInitialOptions(): void {
        this.props.fetchInitialSelectedValues()
            .then(async (options: SelectOption[]) => {
                const enrichedOptions = await this.ensureSelectedValueHasLabel(options);
                this.setState({cachedSelectedOptions: enrichedOptions});
            })
            .catch((e) => {
                this.setState({error: String(e)});
            });
    }

    private async ensureSelectedValueHasLabel(options: SelectOption[]): Promise<SelectOption[]> {
        const stringValue = toValues(this.props.value)[0];

        if (this.props.fieldKey !== TEAM_FIELD || !stringValue || options.some((opt) => opt.value === stringValue)) {
            return options;
        }

        try {
            const allOptions = await this.props.search('');
            const matched = allOptions.find((option) => option.value === stringValue);
            return matched ? [...options, matched] : options;
        } catch (e) {
            this.setState({error: String(e)});
            return options;
        }
    }

    search = async (userInput: string): Promise<SelectOption[]> => {
        try {
            const options = await this.props.search(userInput);
            return options || [];
        } catch (e) {
            this.setState({error: String(e)});
            return [];
        }
    };

    debouncedSearch = debounce(this.search, searchDebounceDelay);

    runSearch = async (userInput: string, debounced: boolean) => {
        const requestID = ++this.latestSearch;
        this.setState({loading: true});

        const options = await (debounced ? this.debouncedSearch(userInput) : this.search(userInput));
        if (requestID !== this.latestSearch) {
            return;
        }

        this.setState((state) => ({
            loading: false,
            options,
            defaultOptions: userInput ? state.defaultOptions : options,
        }));
    };

    handleFocus = () => {
        if (this.state.defaultOptions === null && !this.state.loading) {
            this.runSearch('', false);
        }
    };

    handleInputChange = (query: string) => {
        if (!query && this.state.defaultOptions) {
            this.latestSearch++;
            this.setState((state) => ({loading: false, options: state.defaultOptions || []}));
            return;
        }
        this.runSearch(query, true);
    };

    onChange = (value: string | string[] | null): void => {
        const values = toValues(value);
        const known = [...this.state.options, ...this.state.cachedSelectedOptions];
        const picked = values
            .filter((v) => !this.state.cachedSelectedOptions.some((opt) => opt.value === v))
            .map((v) => known.find((opt) => opt.value === v))
            .filter((opt): opt is SelectOption => Boolean(opt));

        this.latestSearch++;
        this.setState((state) => ({
            loading: false,
            options: state.defaultOptions || state.options,
            cachedSelectedOptions: state.cachedSelectedOptions.concat(picked),
            invalid: this.props.resetInvalidOnChange ? false : state.invalid,
        }));

        if (this.props.isMulti) {
            this.props.onChange(values);
        } else {
            this.props.onChange(values[0] || '');
        }
    };

    isValid = (): boolean => {
        if (!this.props.required) {
            return true;
        }

        const valid = toValues(this.props.value).length > 0;
        this.setState({invalid: !valid});
        return valid;
    };

    getSelectedOptions = (values: string[]): SelectOption[] => {
        return values.map((v) => {
            return this.state.cachedSelectedOptions.find((opt) => opt.value === v) ||
                this.state.options.find((opt) => opt.value === v) ||
                {value: v, label: v};
        });
    };

    render = (): JSX.Element => {
        const {label, required, hideRequiredStar, isMulti} = this.props;
        const values = toValues(this.props.value);
        const ariaLabel = this.props['aria-label'];

        let error: React.ReactNode = null;
        if (this.state.error) {
            error = this.state.error;
        } else if (required && this.state.invalid) {
            error = REQUIRED_MESSAGE;
        }

        let errorId;
        if (error) {
            errorId = `${this.props.name || this.fallbackId}-error`;
        }

        return (
            <Setting
                error={error}
                errorId={errorId}
            >
                <Combobox
                    id={this.props.name}
                    aria-describedby={errorId}
                    label={label && fieldLabel(label, required && !hideRequiredStar)}
                    aria-label={ariaLabel}
                    listboxLabel={typeof label === 'string' ? label : ariaLabel}
                    options={this.state.options}
                    selectedOptions={this.getSelectedOptions(values)}
                    multiple={isMulti}
                    value={isMulti ? values : (values[0] ?? null)}
                    onChange={this.onChange}
                    onFocus={this.handleFocus}
                    onInputChange={this.handleInputChange}
                    filter={false}
                    loading={this.state.loading}
                    clearable={this.props.isClearable}
                    disabled={this.props.isDisabled}
                    invalid={required && this.state.invalid}
                    required={required}
                    placeholder={this.props.placeholder}
                    submitOnEnter={false}
                    zIndex={Z_INDEX_MENU_PORTAL}
                />
            </Setting>
        );
    };
}
