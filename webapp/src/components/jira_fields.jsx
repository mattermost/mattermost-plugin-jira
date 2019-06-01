// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import JiraField from 'components/jira_field';

export default class JiraFields extends React.PureComponent {
    static propTypes = {
        fields: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        values: PropTypes.object,
        allowedFields: PropTypes.object.isRequired,
        allowedSchemaCustom: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
    };

    render() {
        const fields = this.props.fields;
        const {allowedFields, allowedSchemaCustom} = this.props;

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

            // only allow these default Jira fields and custom types until handle further types
            if ((fields[fieldName].schema.custom && !allowedSchemaCustom.includes(fields[fieldName].schema.custom)) ||
                (!fields[fieldName].schema.custom && !allowedFields.includes(fieldName))
            ) {
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
                    theme={this.props.theme}
                />
            );
        });
    }
}
