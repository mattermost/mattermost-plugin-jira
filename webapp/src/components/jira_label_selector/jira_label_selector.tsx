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
    searchLabels: (params: object) => Promise<any>;
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

export default class JiraLabelSelector extends React.PureComponent<Props, State> {
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

    searchLabels = (inputValue: string): Promise<ReactSelectOption[]> => {
        const params = {
            fieldValue: inputValue,
        };
        return this.props.searchLabels(params).then(({data}) => {
            return data.results.map((label) => ({
                value: label.value,
                label: label.value,
            }));
        }).catch((e) => {
            this.setState({error: e});
            return [];
        });
    };

    debouncedSearchLabel = debounce(this.searchLabels, searchDebounceDelay);

    handleLabelSearch = (inputValue: string): Promise<ReactSelectOption[]> => {
        return this.debouncedSearchLabel(inputValue);
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
        const {value} = this.props;
        const {cachedSelectedOptions} = this.state;
        const preloadedLabels = value.map((v) => {
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
                name={'label'}
                value={preloadedLabels}
                onChange={this.onChange}
                loadOptions={this.handleLabelSearch}
                required={this.props.required}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                styles={getStyleForReactSelect(this.props.theme)}
            />
        );

        return (
            <Setting
                {...{}}
                inputId={'label'}
            >
                {selectComponent}
            </Setting>
        );
    }
}