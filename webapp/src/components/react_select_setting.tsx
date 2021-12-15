// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import ReactSelect from 'react-select';
import AsyncSelect, {Props as ReactSelectProps} from 'react-select/async';
import CreatableSelect from 'react-select/creatable';
import {injectIntl, IntlShape} from 'react-intl';

import {Theme} from 'mattermost-redux/types/preferences';

import {ActionMeta, ValueType} from 'react-select/src/types';

import Setting from 'components/setting';

import {getStyleForReactSelect} from 'utils/styles';

import {ReactSelectOption} from 'types/model';

const MAX_NUM_OPTIONS = 100;

type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>

export type Props = Omit<ReactSelectProps<ReactSelectOption>, 'theme'> & {
    theme: Theme;
    addValidate?: (isValid: () => boolean) => void;
    removeValidate?: (isValid: () => boolean) => void;
    allowUserDefinedValue?: boolean;
    limitOptions?: boolean;
    resetInvalidOnChange?: boolean;
    intl: IntlShape;
};

type State = {
    invalid: boolean;
};

export class ReactSelectSetting extends React.PureComponent<Props, State> {
    state: State = {invalid: false};

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

    componentDidUpdate(prevProps: Props, prevState: State) {
        if (prevState.invalid && (this.props.value && this.props.value.value) !== (prevProps.value && prevProps.value.value)) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleChange = (value: ReactSelectOption | ReactSelectOption[], action: ActionMeta) => {
        if (this.props.onChange) {
            if (Array.isArray(value)) {
                this.props.onChange(this.props.name, value.map((x) => x.value));
            } else {
                const newValue = value ? value.value : null;
                this.props.onChange(this.props.name, newValue);
            }
        }
        if (this.props.resetInvalidOnChange) {
            this.setState({invalid: false});
        }
    };

    // Standard search term matching plus reducing to < 100 items
    filterOptions = (input: string) => {
        let options = this.props.options;
        if (input) {
            options = options.filter((opt: ReactSelectOption) => opt.label.toUpperCase().includes(input.toUpperCase()));
        }
        return Promise.resolve(options.slice(0, MAX_NUM_OPTIONS));
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        let valid = Boolean(this.props.value);
        if (this.props.value && Array.isArray(this.props.value)) {
            valid = Boolean(this.props.value.length);
        }

        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        const {formatMessage} = this.props.intl;

        const requiredMsg = formatMessage({defaultMessage: 'This field is required.'});
        let validationError = null;

        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        let selectComponent = null;
        if (this.props.limitOptions && this.props.options.length > MAX_NUM_OPTIONS) {
            // The parent component has let us know that we may have a large number of options, and that
            // the dataset is static. In this case, we use the AsyncSelect component and synchronous func
            // this.filterOptions() to limit the number of options being rendered at a given time.
            selectComponent = (
                <AsyncSelect
                    {...this.props}
                    loadOptions={this.filterOptions}
                    defaultOptions={true}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    onChange={this.handleChange}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
            );
        } else if (this.props.allowUserDefinedValue) {
            selectComponent = (
                <CreatableSelect
                    {...this.props}
                    noOptionsMessage={() => formatMessage({defaultMessage: 'Start typing...'})}
                    formatCreateLabel={(value) => `Add "${value}"`}
                    placeholder=''
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    onChange={this.handleChange}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
            );
        } else {
            selectComponent = (
                <ReactSelect
                    {...this.props}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    onChange={this.handleChange}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
            );
        }
        return (
            <Setting
                inputId={this.props.name}
                {...this.props}
            >
                {selectComponent}
                {validationError}
            </Setting>
        );
    }
}

export default injectIntl(ReactSelectSetting);
