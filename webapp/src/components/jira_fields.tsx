// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import {CreateIssueFields, IssueMetadata, JiraField} from 'types/model';

import JiraFieldComponent, {isFieldSupported} from './jira_field';

type Props = {
    fields: {[key: string]: JiraField};
    instanceID: string;
    onChange: (key: string, value: JiraField) => void;
    issueMetadata: IssueMetadata | null;
    values: CreateIssueFields;
    isFilter: boolean;
    theme: Theme;
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;
}

export default class JiraFields extends React.Component<Props> {
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
        theme: PropTypes.object.isRequired,
        addValidate: PropTypes.func.isRequired,
        removeValidate: PropTypes.func.isRequired,
    };

    getSortedFields = () => {
        const {fields} = this.props;
        let fieldKeys = Object.keys(fields);

        const start = [];
        if (fieldKeys.includes('summary')) {
            start.push('summary');
        }
        if (fieldKeys.includes('description')) {
            start.push('description');
        }

        fieldKeys = fieldKeys.filter((key) => {
            if (['summary', 'description', 'issuetype', 'project'].includes(key)) {
                return false;
            }

            return isFieldSupported(fields[key]);
        }).sort((a, b) => {
            const f1 = fields[a];
            const f2 = fields[b];

            if (f1.required && !f2.required) {
                return -1;
            }
            if (!f1.required && f2.required) {
                return 1;
            }
            return fields[a].name > fields[b].name ? 1 : -1;
        });

        return start.concat(fieldKeys);
    };

    render() {
        const {fields, values} = this.props;

        if (!fields) {
            return null;
        }

        let projectKey: string | undefined;
        if (values && values.project) {
            projectKey = values.project.key;
        }

        const keys = this.getSortedFields();
        return keys.map((key) => {
            const field = fields[key];
            return (
                <JiraFieldComponent
                    instanceID={this.props.instanceID}
                    key={key}
                    id={key}
                    issueMetadata={this.props.issueMetadata}
                    projectKey={projectKey}
                    field={field}
                    obeyRequired={true}
                    onChange={this.props.onChange}
                    value={this.props.values && this.props.values[key]}
                    isFilter={this.props.isFilter}
                    theme={this.props.theme}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                />
            );
        });
    }
}
