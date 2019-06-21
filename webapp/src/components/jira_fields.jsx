// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import JiraField from 'components/jira_field';

export default class JiraFields extends React.Component {
    static propTypes = {
        fields: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        values: PropTypes.object,
        isFilter: PropTypes.bool,
        allowedFields: PropTypes.object.isRequired,
        allowedSchemaCustom: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
    };

    render() {
        const {allowedFields, allowedSchemaCustom, fields} = this.props;

        if (!fields) {
            return null;
        }

        let fieldNames = Object.keys(fields);
        const fullLength = fieldNames.length;
        fieldNames = fieldNames.filter((name) => name !== 'summary');
        if (fullLength > fieldNames.length) {
            fieldNames.unshift('summary');
        }

        return fieldNames.map((fieldName) => {
            // Always Required Jira fields
            if (fieldName === 'project' || fieldName === 'issuetype') {
                return null;
            }

            // only allow these some custom types until handle further types
            if (fields[fieldName].schema.custom && !allowedSchemaCustom.includes(fields[fieldName].schema.custom)) {
                return null;
            }

            // only allow some default Jira fields until handle further types
            if (!fields[fieldName].schema.custom && !allowedFields.includes(fieldName)) {
                return null;
            }

            return (
                <JiraField
                    key={fieldName}
                    id={fieldName}
                    field={fields[fieldName]}
                    obeyRequired={true}
                    onChange={this.props.onChange}
                    value={this.props.values && this.props.values[fieldName]}
                    isFilter={this.props.isFilter}
                    theme={this.props.theme}
                />
            );
        });
    }
}
