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
        fieldKey: PropTypes.string.isRequired,
        obeyRequired: PropTypes.bool,
        onChange: PropTypes.func.isRequired,
        value: PropTypes.any,
        isFilter: PropTypes.bool,
    };

    static defaultProps = {
        obeyRequired: true,
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
    };

    renderCreateFields() {
        const {field, fieldKey, obeyRequired} = this.props;

        if (field.schema.system === 'description') {
            return (
                <Input
                    key={this.props.id}
                    id={this.props.id}
                    label={field.name}
                    type='textarea'
                    onChange={this.props.onChange}
                    required={this.props.obeyRequired && field.required}
                    value={this.props.value}
                />
            );
        }

        if (field.schema.type === 'string') {
            return (
                <Input
                    key={this.props.id}
                    id={this.props.id}
                    label={field.name}
                    type='input'
                    onChange={this.props.onChange}
                    required={this.props.obeyRequired && field.required}
                    value={this.props.value}
                />
            );
        }

        // if this.props.field has allowedValues, then props.value will be an object
        if (field.allowedValues && field.allowedValues.length) {
            const options = field.allowedValues.map(this.makeReactSelectValue);

            return (
                <ReactSelectSetting
                    key={this.props.id}
                    name={this.props.id}
                    label={field.name}
                    options={options}
                    required={this.props.obeyRequired && field.required}
                    onChange={(id, val) => this.props.onChange(id, {id: val})}
                    isMulti={false}
                    value={options.find((option) => option.value === this.props.value)}
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
