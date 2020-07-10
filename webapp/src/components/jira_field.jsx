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

    makeReactSelectValue = (allowedValue) => {
        const label = allowedValue.name ? allowedValue.name : allowedValue.value;
        return (
            {value: allowedValue.id, label, allowedValue}
        );
    };

    renderCreateFields() {
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
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
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
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                />
            );
        }

        const selectProps = {
            theme: this.props.theme,
            addValidate: this.props.addValidate,
            removeValidate: this.props.removeValidate,
            label: field.name,
            required: this.props.obeyRequired && field.required,
            hideRequiredStar: false,
            resetInvalidOnChange: true,
            placeholder: '',
            isClearable: true,
        }

        if (field.schema.custom === 'com.pyxis.greenhopper.jira:gh-epic-link') {
            return (
                <JiraEpicSelector
                    {...selectProps}
                    key={this.props.id}
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
                    key={this.props.id}
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
                    key={this.props.id}
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
                    key={this.props.id}
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
                    value={options.find((option) => option.value === (this.props.value && this.props.value.id))}
                    theme={this.props.theme}
                    isClearable={true}
                    components={{Option: JiraField.IconOption}}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
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
                    onChange={(id, val) => this.props.onChange(id, {id: val})}
                    isMulti={true}
                    value={value}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
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
