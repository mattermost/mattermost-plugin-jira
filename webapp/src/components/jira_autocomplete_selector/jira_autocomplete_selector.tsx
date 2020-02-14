// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import AsyncSelect from 'react-select/async';
import debounce from 'debounce-promise';

import Setting from 'components/setting';
import {getStyleForReactSelect} from 'utils/styles';
import {ReactSelectOption} from 'types/model';

const searchDebounceDelay = 400;

type Props = {
    required?: boolean;
    hideRequiredStar?: boolean;
    theme: object;
    searchAutoCompleteFields: (params: object) => Promise<any>;
    fieldName: string;
    onChange: (values: string[]) => void;
    value: string[];
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
    resetInvalidOnChange?: boolean;
};

type State = {
    invalid: boolean;
    error?: string;
    cachedSelectedOptions: ReactSelectOption[];
};

export default class JiraAutoCompleteSelector extends React.PureComponent<Props, State> {
    state = {
        invalid: false,
        cachedSelectedOptions: [],
    }

    componentDidMount(): void {
        if (this.props.addValidate) {
            this.props.addValidate(this.isValid);
        }
    }

    componentWillUnmount(): void {
        if (this.props.removeValidate) {
            this.props.removeValidate(this.isValid);
        }
    }

    isValid = (): boolean => {
        if (!this.props.required) {
            return true;
        }

        const valid = this.props.value && this.props.value.toString().length !== 0;
        this.setState({invalid: !valid});
        return valid;
    };

    searchAutoCompleteFields = (inputValue: string): Promise<ReactSelectOption[]> => {
        const {fieldName} = this.props;
        const params = {
            fieldValue: inputValue,
            fieldName,
        };
        return this.props.searchAutoCompleteFields(params).then(({data}) => {
            return data.results.map((label) => ({
                value: label.value,
                label: label.value,
            }));
        }).catch((e) => {
            this.setState({error: e});
            return [];
        });
    };

    debouncedSearch = debounce(this.searchAutoCompleteFields, searchDebounceDelay);

    handleSearch = (inputValue: string): Promise<ReactSelectOption[]> => {
        return this.debouncedSearch(inputValue);
    }

    onChange = (options: ReactSelectOption[]): void => {
        if (!options) {
            this.props.onChange([]);
            return;
        }
        this.setState({cachedSelectedOptions: this.state.cachedSelectedOptions.concat(options)});
        this.props.onChange(options.map((v) => v.value));

        if (this.props.resetInvalidOnChange) {
            this.setState({invalid: false});
        }
    }

    render = (): JSX.Element => {
        const {value, fieldName} = this.props;
        const {cachedSelectedOptions} = this.state;
        const preloadedFields = value.map((v) => {
            if (cachedSelectedOptions.length > 0) {
                const alreadySelected = cachedSelectedOptions.find((option) => option.value === v);
                if (alreadySelected) {
                    return alreadySelected;
                }
            }
            return {
                label: v,
                value: v,
            };
        });

        const selectComponent = (
            <AsyncSelect
                isMulti={true}
                name={fieldName}
                value={preloadedFields}
                onChange={this.onChange}
                loadOptions={this.handleSearch}
                required={this.props.required}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                styles={getStyleForReactSelect(this.props.theme)}
            />
        );

        return (
            <Setting
                {...{}}
                inputId={fieldName}
            >
                {selectComponent}
            </Setting>
        );
    }
}