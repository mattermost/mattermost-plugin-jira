// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import ReactSelectSetting from 'components/react_select_setting';
import Input from 'components/input';

export default class JiraField extends React.PureComponent {
    static propTypes = {
        id: PropTypes.object.isRequired,
        field: PropTypes.object.isRequired,
        obeyRequired: PropTypes.bool,
        onChange: PropTypes.func.isRequired,
        value: PropTypes.any,
        theme: PropTypes.object.isRequired,
    };

    static defaultProps = {
        obeyRequired: true,
    };

    // Creates an option for react-select from an allowedValue from the jira field metadata
    // includes both .value and .name because allowedValue test cases have used .value or .name
    // and are mutually exclusive.
    // should wrap this with some if else logic
    makeReactSelectValue = (allowedValue) => {
        const iconLabel = (
            <React.Fragment>
                <img
                    style={getStyle().jiraIcon}
                    src={allowedValue.iconUrl}
                />
                {allowedValue.value}
                {allowedValue.name}
            </React.Fragment>
        );
        return (
            {value: allowedValue.id, label: iconLabel}
        );
    };

    render() {
        const field = this.props.field;

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

        // detect if JIRA multiline textarea, and set for JiraField component
        if (field.schema.custom === 'com.atlassian.jira.plugin.system.customfieldtypes:textarea') {
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
        if (field.allowedValues && field.allowedValues.length && field.schema.type !== 'array') {
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
                    theme={this.props.theme}
                    isClearable={true}
                />
            );
        }
        return null;
    }
}

const getStyle = () => ({
    jiraIcon: {
        height: '16px',
        marginRight: '5px',
    },
});
