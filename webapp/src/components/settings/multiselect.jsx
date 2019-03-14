// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import ReactSelect from 'react-select';

import Setting from './setting.jsx';

export default class MultiSelectSetting extends PureComponent {
    static propTypes = {
        id: PropTypes.string.isRequired,
        options: PropTypes.array.isRequired,
        label: PropTypes.node.isRequired,
        selected: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
        disabled: PropTypes.bool,
        required: PropTypes.bool,
        helpText: PropTypes.node,
        noResultText: PropTypes.node,// ?
        errorText: PropTypes.node, // ?
        notPresent: PropTypes.node,// ?
    };

    static defaultProps = {
        disabled: false,
        required: false
    };

    handleChange = (newValue) => {
        const values = newValue.map((n) => {
            return {name: n.value};
        });

        this.props.onChange(this.props.id, values);
    };

    render() {
        return (
            <Setting
                label={this.props.label}
                inputId={this.props.id}
                helpText={this.props.helpText}
                required={this.props.required}
            >
                <ReactSelect
                    id={this.props.id}
                    multi={true}
                    labelKey='text'
                    options={this.props.options}
                    joinValues={true}
                    clearable={false}
                    disabled={this.props.disabled}
                    noResultsText={this.props.noResultText}
                    onChange={this.handleChange}
                    value={this.props.selected}
                />
            </Setting>
        );

    }
}