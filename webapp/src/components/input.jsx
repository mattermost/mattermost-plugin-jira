// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import Setting from './setting.jsx';

export default class Input extends PureComponent {
    static propTypes = {
        id: PropTypes.string.isRequired,
        label: PropTypes.node.isRequired,
        placeholder: PropTypes.string,
        helpText: PropTypes.node,
        value: PropTypes.oneOfType([
            PropTypes.string,
            PropTypes.number,
        ]).isRequired,
        maxLength: PropTypes.number,
        onChange: PropTypes.func,
        disabled: PropTypes.bool,
        required: PropTypes.bool,
        type: PropTypes.oneOf([
            'number',
            'input',
            'textarea',
        ]),
    };

    static defaultProps = {
        type: 'input',
        maxLength: null,
        required: false,
    };

    handleChange = (e) => {
        if (this.props.type === 'number') {
            this.props.onChange(this.props.id, parseInt(e.target.value, 10));
        } else {
            this.props.onChange(this.props.id, e.target.value);
        }
    };

    render() {
        let input = null;
        if (this.props.type === 'input') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='text'
                    placeholder={this.props.placeholder}
                    value={this.props.value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    required={this.props.required}
                />
            );
        } else if (this.props.type === 'number') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='number'
                    placeholder={this.props.placeholder}
                    value={this.props.value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    required={this.props.required}
                />
            );
        } else if (this.props.type === 'textarea') {
            input = (
                <textarea
                    id={this.props.id}
                    className='form-control asd'
                    rows='5'
                    readOnly
                    placeholder={this.props.placeholder}
                    value={this.props.value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    required={this.props.required}
                />
            );
        }

        return (
            <Setting
                label={this.props.label}
                helpText={this.props.helpText}
                inputId={this.props.id}
                required={this.props.required}
            >
                {input}
            </Setting>
        );
    }
}
