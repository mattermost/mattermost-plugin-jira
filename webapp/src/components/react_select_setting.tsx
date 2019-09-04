// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import AsyncSelect from 'react-select/async';

import Setting from 'components/setting';
import {getStyleForReactSelect} from 'utils/styles';

const MAX_NUM_OPTIONS = 100;

type Props = {
    name: string;
    onChange?: (name: string, value: any) => void;
    theme: object;
    isClearable?: boolean;
    options: any[];
    value?: any;
    required?: boolean;
    isMulti?: boolean;
    label?: string;
    components?: any;
};

type State = {
    invalid: boolean;
};

export default class ReactSelectSetting extends React.PureComponent<Props, State> {
    constructor(props: Props) {
        super(props);

        this.state = {invalid: false};
    }

    componentDidUpdate(prevProps: Props, prevState: State) {
        if (prevState.invalid && (this.props.value && this.props.value.value) !== (prevProps.value && prevProps.value.value)) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleChange = (value: any) => {
        if (this.props.onChange) {
            if (Array.isArray(value)) {
                this.props.onChange(this.props.name, value.map((x) => x.value));
            } else {
                const newValue = value ? value.value : null;
                this.props.onChange(this.props.name, newValue);
            }
        }
    };

    // Standard search term matching plus reducing to < 100 items
    filterOptions = (input: string): Promise<any[]> => {
        let options = this.props.options;
        if (input) {
            options = options.filter((x: any) => x.label.toUpperCase().includes(input.toUpperCase()));
        }
        return Promise.resolve(options.slice(0, MAX_NUM_OPTIONS));
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }
        const valid = Boolean(this.props.value);
        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        const requiredMsg = 'This field is required.';
        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        return (
            <Setting
                inputId={this.props.name}
                {...this.props}
            >
                <AsyncSelect
                    name={this.props.name}
                    isClearable={this.props.isClearable}
                    options={this.props.options}
                    value={this.props.value}
                    required={this.props.required}
                    isMulti={this.props.isMulti}
                    label={this.props.label}
                    components={this.props.components}
                    loadOptions={this.filterOptions}
                    defaultOptions={true}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    onChange={this.handleChange}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {validationError}
            </Setting>
        );
    }
}
