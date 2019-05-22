// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import ReactSelectSetting from 'components/react_select_setting';
import Input from 'components/input';

export default class JiraField extends React.Component {
    static propTypes = {
        id: PropTypes.object.isRequired,
        field: PropTypes.object.isRequired,
        obeyRequired: PropTypes.bool,
        onChange: PropTypes.func,
        value: PropTypes.any,
        isFilter: PropTypes.bool,
    };

    static defaultProps = {
        obeyRequired: true,
    };

    handleChange = (id, value) => {
        if (this.props.onChange) {
            this.props.onChange(id, value);
        }
    };

    // Creates an option for react-select from an allowedValue from the jira field metadata
    makeReactSelectValue = (allowedValue) => {
        const iconLabel = (
            <React.Fragment>
                <img
                    style={getStyle().jiraIcon}
                    src={allowedValue.iconUrl}
                />
                {allowedValue.name}
            </React.Fragment>
        );
        return (
            {value: allowedValue.id, label: iconLabel}
        );
    }

    renderCreateFields() {
        const field = this.props.field;

        if (field.schema.system === 'description') {
            return (
                <Input
                    key={field.key}
                    id={field.key}
                    label={field.name}
                    type='textarea'
                    onChange={this.handleChange}
                    required={this.props.obeyRequired && field.required}
                    value={this.props.value}
                />
            );
        }

        if (field.schema.type === 'string') {
            return (
                <Input
                    key={field.key}
                    id={field.key}
                    label={field.name}
                    type='input'
                    onChange={this.handleChange}
                    required={this.props.obeyRequired && field.required}
                    value={this.props.value}
                />
            );
        }

        if (field.allowedValues && field.allowedValues.length) {
            const options = field.allowedValues.map(this.makeReactSelectValue);
            return (
                <ReactSelectSetting
                    key={field.key}
                    name={field.key}
                    label={field.name}
                    options={options}
                    required={this.props.obeyRequired && field.required}
                    onChange={this.handleChange}
                    isMulti={false}
                    value={options.filter((option) => option.value === this.props.value)}
                />
            );
        }
        return null;
    }

    renderFilterFields() {
        const field = this.props.field;

        if (field.allowedValues && field.allowedValues.length) {
            const options = field.allowedValues.map(this.makeReactSelectValue);
            let value;
            if (this.props.value) {
                value = options.filter((option) => this.props.value.includes(option.value));
            }

            return (
                <ReactSelectSetting
                    key={field.key}
                    name={field.key}
                    label={field.name}
                    options={options}
                    required={this.props.obeyRequired && field.required}
                    onChange={this.handleChange}
                    isMulti={true}
                    value={value}
                />
            );
        }

        return null;
    }

    render() {
        if (this.props.isFilter) {
            return this.renderFilterFields();
        }

        return this.renderCreateFields();
    }
}

const getStyle = () => ({
    jiraIcon: {
        height: '16px',
        marginRight: '5px',
    },
});
