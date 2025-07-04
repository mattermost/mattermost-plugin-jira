// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import {IssueMetadata, ReactSelectOption} from 'types/model';

import {getStyleForReactSelect} from 'utils/styles';

import {Props as ReactSelectSettingProps} from 'components/react_select_setting';
import Setting from 'components/setting';
import {TEAM_FIELD} from 'constant';

const searchDebounceDelay = 400;

export type Props = ReactSelectSettingProps & {
    hideRequiredStar?: boolean;
    onChange: (values: string | string[]) => void;
    value?: string | string[];
    fetchInitialSelectedValues: () => Promise<ReactSelectOption[]>;
    search: (searchTerm: string) => Promise<ReactSelectOption[]>;
    theme: Theme;
    isMulti?: boolean;
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    issueMetadata: IssueMetadata;
    resetInvalidOnChange?: boolean;
    instanceID: string;
    fieldKey?: string;
};

type State = {
    invalid: boolean;
    error?: string;
    cachedSelectedOptions: ReactSelectOption[];
};

export default class BackendSelector extends React.PureComponent<Props, State> {
    static defaultProps = {
        isMulti: false,
    };

    constructor(props: Props) {
        super(props);

        this.state = {
            invalid: false,
            error: '',
            cachedSelectedOptions: [],
        };
    }

    componentDidMount(): void {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }

        this.loadInitialOptions();
    }

    private loadInitialOptions(): void {
        this.props.fetchInitialSelectedValues()
            .then(async (options: ReactSelectOption[]) => {
                const enrichedOptions = await this.ensureSelectedValueHasLabel(options);
                this.setState({cachedSelectedOptions: enrichedOptions});
            })
            .catch((e) => {
                this.setState({error: e});
            });
    }

    private async ensureSelectedValueHasLabel(options: ReactSelectOption[]): Promise<ReactSelectOption[]> {
        const value = this.props.value;
        const stringValue = Array.isArray(value) ? value[0] : String(value ?? '');

        if (!(this.props.fieldKey === TEAM_FIELD) || !stringValue || options.some((opt) => opt.value === stringValue)) {
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

    componentWillUnmount(): void {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    componentDidUpdate(prevProps: Props, prevState: State): void {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleIssueSearchTermChange = (inputValue: string): Promise<ReactSelectOption[]> => {
        return this.debouncedSearch(inputValue);
    };

    search = async (userInput: string): Promise<ReactSelectOption[]> => {
        return this.props.search(userInput).then((options) => {
            return options || [];
        }).catch((e) => {
            this.setState({error: e});
            return [];
        });
    };

    debouncedSearch = debounce(this.search, searchDebounceDelay);

    onChange = (options: ReactSelectOption | ReactSelectOption[]): void => {
        if (!options) {
            if (this.props.isMulti) {
                this.props.onChange([]);
            } else {
                this.props.onChange('');
            }
            return;
        }
        this.setState({cachedSelectedOptions: this.state.cachedSelectedOptions.concat(options)});
        if (this.props.isMulti) {
            this.props.onChange((options as ReactSelectOption[]).map((v) => v.value));
        } else {
            this.props.onChange((options as ReactSelectOption).value);
        }

        if (this.props.resetInvalidOnChange) {
            this.setState({invalid: false});
        }
    };

    isValid = (): boolean => {
        if (!this.props.required) {
            return true;
        }

        const valid = Boolean(this.props.value && this.props.value.toString().length !== 0);
        this.setState({invalid: !valid});
        return valid;
    };

    render = (): JSX.Element => {
        const serverError = this.state.error;
        let errComponent;
        if (serverError) {
            errComponent = (
                <p className='alert alert-danger'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span> {serverError.toString()}</span>
                </p>
            );
        }

        const requiredMsg = 'This field is required.';
        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        let value;
        const valueToOption = (v: string): ReactSelectOption => {
            // Ensure the value is always a string for comparison.
            let stringValue: string;
            if (typeof v === 'string') {
                stringValue = v;
            } else if (Array.isArray(v)) {
                stringValue = v[0];
            } else {
                stringValue = String(v);
            }

            if (this.state.cachedSelectedOptions && this.state.cachedSelectedOptions.length) {
                const selected = this.state.cachedSelectedOptions.find((option) => option.value === stringValue);
                if (selected) {
                    return selected;
                }
            }

            return {
                label: stringValue,
                value: stringValue,
            };
        };

        if (this.props.isMulti) {
            value = (this.props.value as string[]).map(valueToOption);
        } else if (this.props.value) {
            value = valueToOption(this.props.value as string);
        }

        const selectComponent = (
            <AsyncSelect
                {...this.props}
                value={value}
                onChange={this.onChange}
                required={this.props.required}
                isMulti={this.props.isMulti}
                defaultOptions={true}
                loadOptions={this.handleIssueSearchTermChange}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                styles={getStyleForReactSelect(this.props.theme)}
            />
        );

        return (
            <Setting
                {...this.props}
                inputId={'epic'}
            >
                {selectComponent}
                {errComponent}
                {validationError}
            </Setting>
        );
    };
}
