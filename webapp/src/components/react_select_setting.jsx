// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import ReactSelect from 'react-select';

import Setting from 'components/setting';

import {getStyleForReactSelect} from 'utils/styles';

export default class ReactSelectSetting extends React.PureComponent {
    static propTypes = {
        name: PropTypes.string.isRequired,
        onChange: PropTypes.func,
        theme: PropTypes.object.isRequired,
        isClearable: PropTypes.bool,
        value: PropTypes.object,
        required: PropTypes.bool,
    };

    constructor(props) {
        super(props);

        this.state = {invalidRequired: false};
    }

    componentDidUpdate(prevProps, prevState) {
        if (prevState.invalidRequired && this.props.value !== prevProps.value) {
            this.setState({invalidRequired: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleChange = (value) => {
        if (this.props.onChange) {
            const newValue = value ? value.value : null;
            this.props.onChange(this.props.name, newValue);
        }
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }
        const valid = this.props.value && this.props.value.toString().length !== 0;
        this.setState({invalidRequired: !valid});
        return valid;
    };

    render() {
        const requiredMsg = 'This field is required.';
        let error = null;
        if (this.props.required && this.state.invalidRequired) {
            error = (
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
                <ReactSelect
                    {...this.props}
                    ref={this.ref}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    onChange={this.handleChange}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
                {error}
            </Setting>
        );
    }
}
