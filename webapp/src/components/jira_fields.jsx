// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import JiraField from 'components/jira_field';

export default class JiraFields extends React.Component {
    static propTypes = {
        fields: PropTypes.object.isRequired,
        onChange: PropTypes.func,
        values: PropTypes.object,
        isFilter: PropTypes.bool,
    };

    render() {
        if (!this.props.fields) {
            return null;
        }

        let fieldNames = Object.keys(this.props.fields);
        const fullLength = fieldNames.length;
        fieldNames = fieldNames.filter((name) => name !== 'summary');
        if (fullLength > fieldNames.length) {
            fieldNames.unshift('summary');
        }

        return fieldNames.map((fieldName) => {
            if (fieldName === 'project' || fieldName === 'issuetype' || fieldName === 'reporter' || (fieldName !== 'description' && !this.props.fields[fieldName].required)) {
                return null;
            }
            return (
                <JiraField
                    key={fieldName}
                    id={fieldName}
                    field={this.props.fields[fieldName]}
                    obeyRequired={true}
                    onChange={this.props.onChange}
                    value={this.props.values && this.props.values[fieldName]}
                    isFilter={this.props.isFilter}
                />
            );
        });
    }
}
