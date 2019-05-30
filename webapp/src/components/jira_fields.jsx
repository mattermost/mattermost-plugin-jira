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
        theme: PropTypes.object.isRequired,
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
            if (fieldName === 'project' || fieldName === 'issuetype') {
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
                    theme={this.props.theme}
                />
            );
        });
    }
}
