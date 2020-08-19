// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import JiraField from 'components/jira_field';

export default class JiraFields extends React.Component {
    static propTypes = {
        fields: PropTypes.oneOfType([
            PropTypes.object,
            PropTypes.array,
        ]).isRequired,
        instanceID: PropTypes.string.isRequired,
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
        const {allowedFields, allowedSchemaCustom, fields} = this.props;
        let fieldKeys = Object.keys(fields);

        const start = [];
        const summary = fieldKeys.find((key) => key === 'summary');
        if (summary) {
            start.push(summary);
        }
        const description = fieldKeys.find((key) => key === 'description');
        if (description) {
            start.push(description);
        }

        fieldKeys = fieldKeys.filter((key) => {
            const field = fields[key];
            if (['summary', 'description', 'issuetype', 'project'].includes(key)) {
                return false;
            }
            if (field.schema.custom && !allowedSchemaCustom.includes(field.schema.custom)) {
                return false;
            }
            if (!field.schema.custom && !allowedFields.includes(key)) {
                return false;
            }

            return true;
        }).sort((a, b) => {
            return a.name > b.name ? 1 : -1;
        });

        return start.concat(fieldKeys);
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

        const keys = this.getSortedFields();
        return keys.map((key) => {
            const field = fields[key];
            return (
                <JiraField
                    instanceID={this.props.instanceID}
                    key={key}
                    id={key}
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
            );
        });
    }
}
