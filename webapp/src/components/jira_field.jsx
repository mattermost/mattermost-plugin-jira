// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import {components} from 'react-select';

import ReactSelectSetting from 'components/react_select_setting';
import Input from 'components/input';

import JiraEpicSelector from './data_selectors/jira_epic_selector';
import JiraAutoCompleteSelector from './data_selectors/jira_autocomplete_selector';
import JiraUserSelector from './data_selectors/jira_user_selector';

export default class JiraField extends React.Component {
    static propTypes = {
        id: PropTypes.string.isRequired,
        field: PropTypes.object.isRequired,
        projectKey: PropTypes.string.isRequired,
        issueMetadata: PropTypes.object.isRequired,
        obeyRequired: PropTypes.bool,
        onChange: PropTypes.func.isRequired,
        value: PropTypes.any,
        isFilter: PropTypes.bool,
        theme: PropTypes.object.isRequired,
        addValidate: PropTypes.func.isRequired,
        removeValidate: PropTypes.func.isRequired,
    };

    static defaultProps = {
        obeyRequired: true,
    };

    static IconOption = (props) => {
        let img = null;
        if (props.data.allowedValue.iconUrl) {
            img = (
                <img
                    style={getStyle().jiraIcon}
                    src={props.data.allowedValue.iconUrl}
                />
            );
        }
        return (
            <components.Option
                {...props}
                style={getStyle().selectComponent}
            >
                {img}
                {props.data.label}
            </components.Option>
        );
    };

    renderCreateFields() {
        const field = this.props.field;

        if (field.schema.system === 'description') {
            return (
                <Input
                    id={this.props.id}
                    label={field.name}
                    type='textarea'
                    onChange={this.props.onChange}
                    required={this.props.obeyRequired && field.required}
                    value={this.props.value}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                />
            );
        }

        // detect if JIRA multiline textarea, and set for JiraField component
        if (field.schema.custom === 'com.atlassian.jira.plugin.system.customfieldtypes:textarea') {
            return (
                <Input
                    id={this.props.id}
                    label={field.name}
                    type='textarea'
                    onChange={this.props.onChange}
                    required={this.props.obeyRequired && field.required}
                    value={this.props.value}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                />
            );
        }

        const selectProps = {
            instanceID: this.props.instanceID,
            theme: this.props.theme,
            addValidate: this.props.addValidate,
            removeValidate: this.props.removeValidate,
            label: field.name,
            required: this.props.obeyRequired && field.required,
            hideRequiredStar: false,
            resetInvalidOnChange: true,
            placeholder: '',
            isClearable: true,
        };

        if (field.schema.custom === 'com.pyxis.greenhopper.jira:gh-epic-link') {
            return (
                <JiraEpicSelector
                    {...selectProps}
                    issueMetadata={this.props.issueMetadata}
                    onChange={(value) => {
                        this.props.onChange(this.props.id, value);
                    }}
                    value={this.props.value}
                    isMulti={false}
                />
            );
        }

        if (field.schema.system === 'labels' || field.schema.custom === 'com.atlassian.jira.plugin.system.customfieldtypes:labels') {
            return (
                <JiraAutoCompleteSelector
                    {...selectProps}
                    fieldName={field.name}
                    onChange={(value) => {
                        this.props.onChange(this.props.id, value);
                    }}
                    value={this.props.value || []}
                    isMulti={field.schema.type === 'array'}
                />
            );
        }

        if (field.schema.type === 'user') {
            return (
                <JiraUserSelector
                    {...selectProps}
                    projectKey={this.props.projectKey}
                    fieldName={field.name}
                    onChange={(value) => {
                        this.props.onChange(this.props.id, {accountId: value});
                    }}
                    value={this.props.value && this.props.value.accountId}
                    isMulti={false}
                />
            );
        }

        if (field.schema.type === 'string') {
            return (
                <Input
                    id={this.props.id}
                    label={field.name}
                    type='input'
                    onChange={this.props.onChange}
                    required={this.props.obeyRequired && field.required}
                    value={this.props.value}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                />
            );
        }

        if (field.allowedValues && field.allowedValues.length) {
            const options = field.allowedValues.map((allowedValue) => {
                const label = allowedValue.name ? allowedValue.name : allowedValue.value;
                return (
                    {value: allowedValue.id, label, allowedValue}
                );
            });

            if (field.schema.type === 'array') {
                let selectedOptions = [];
                if (this.props.value) {
                    const values = this.props.value.map((v) => v.id);
                    selectedOptions = options.filter((opt) => values.includes(opt.value));
                }

                const onChange = (id, val) => {
                    const newValue = val.map((v) => ({id: v}));
                    this.props.onChange(id, newValue);
                };

                return (
                    <ReactSelectSetting
                        {...selectProps}
                        name={this.props.id}
                        options={options}
                        onChange={onChange}
                        isMulti={true}
                        value={selectedOptions}
                        components={{Option: JiraField.IconOption}}
                    />
                );
            }
            return (
                <ReactSelectSetting
                    {...selectProps}
                    name={this.props.id}
                    options={options}
                    onChange={(id, val) => this.props.onChange(id, {id: val})}
                    isMulti={false}
                    value={options.find((option) => option.value === (this.props.value && this.props.value.id))}
                    components={{Option: JiraField.IconOption}}
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
