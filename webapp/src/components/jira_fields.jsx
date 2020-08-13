// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import JiraField from 'components/jira_field';
import {isTextField} from 'utils/jira_issue_metadata';

export default class JiraFields extends React.Component {
    static propTypes = {
        fields: PropTypes.oneOfType([
            PropTypes.object,
            PropTypes.array,
        ]).isRequired,
        onChange: PropTypes.func.isRequired,
        issueMetadata: PropTypes.object.isRequired,
        values: PropTypes.object,
        isFilter: PropTypes.bool,
        allowedFields: PropTypes.array.isRequired,
        allowedSchemaCustom: PropTypes.array.isRequired,
        theme: PropTypes.object.isRequired,
        addValidate: PropTypes.func.isRequired,
        removeValidate: PropTypes.func.isRequired,
    };

    getSortedFields = () => {
        const {allowedFields, allowedSchemaCustom} = this.props;
        let fields = Object.values(this.props.fields);

        const start = [];
        const summary = fields.find((f) => f.key === 'summary');
        if (summary) {
            start.push(summary);
        }
        const description = fields.find((f) => f.key === 'description');
        if (description) {
            start.push(description);
        }

        fields = fields.filter((field) => {
            if (['summary', 'description', 'issuetype', 'project'].includes(field.key)) {
                return false;
            }
            if (field.schema.custom && !allowedSchemaCustom.includes(field.schema.custom)) {
                return false;
            }
            if (!field.schema.custom && !allowedFields.includes(field.key)) {
                return false;
            }

            return true;
        }).sort((a, b) => {
            return a.name > b.name ? 1 : -1;
        });

        const selectFields = fields.filter((f) => !isTextField(f));
        const textFields = fields.filter((f) => isTextField(f));

        return [...start, ...selectFields, ...textFields];
    }

    render() {
        const {fields, values} = this.props;

        if (!fields) {
            return null;
        }

        let projectKey;
        if (values && values.project) {
            projectKey = values.project.key;
        }

        return this.getSortedFields().map((field) => (
            <JiraField
                instanceID={this.props.instanceID}
                key={field.key}
                id={field.key}
                issueMetadata={this.props.issueMetadata}
                projectKey={projectKey}
                field={field}
                obeyRequired={true}
                onChange={this.props.onChange}
                value={this.props.values && this.props.values[field.key]}
                isFilter={this.props.isFilter}
                theme={this.props.theme}
                addValidate={this.props.addValidate}
                removeValidate={this.props.removeValidate}
            />
        ));
    }
}
