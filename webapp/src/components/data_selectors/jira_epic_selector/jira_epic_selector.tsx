// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {isEpicNameField, isEpicIssueType} from 'utils/jira_issue_metadata';
import {IssueMetadata, ReactSelectOption, JiraIssue} from 'types/model';

import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';

const searchDefaults = 'ORDER BY updated DESC';

type Props = BackendSelectorProps & {
    searchIssues: (params: {
        jql: string;
        fields: string;
        q: string;
    }) => Promise<{data: JiraIssue[]}>;
    issueMetadata: IssueMetadata;
};

export default class JiraEpicSelector extends React.PureComponent<Props> {
    fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        if (!this.props.value || (this.props.isMulti && !this.props.value.length)) {
            return [];
        }

        let epicIds = '';
        if (this.props.isMulti) {
            epicIds = (this.props.value as string[]).join(', ');
        } else if (this.props.value) {
            epicIds = this.props.value as string;
        }
        const searchStr = `and id IN (${epicIds})`;
        const userInput = ''; // Fetching by saved ids, no user input to process

        return this.fetchEpicsFromJql(searchStr, userInput).then((options) => {
            if (options) {
                return options;
            }
            return [];
        });
    };

    searchIssues = async (userInput: string): Promise<ReactSelectOption[]> => {
        const epicIssueType = this.props.issueMetadata.projects[0].issuetypes.find(isEpicIssueType);
        if (!epicIssueType) {
            return [];
        }

        const epicNameTypeId = Object.keys(epicIssueType.fields).find((key) => isEpicNameField(epicIssueType.fields[key]));
        if (!epicNameTypeId) {
            return [];
        }

        const epicNameTypeName = epicIssueType.fields[epicNameTypeId].name;

        let searchStr = '';
        if (userInput) {
            const cleanedInput = userInput.trim().replace(/"/g, '\\"');
            searchStr = ` and ("${epicNameTypeName}"~"${cleanedInput}" or "${epicNameTypeName}"~"${cleanedInput}*")`;
        }

        return this.fetchEpicsFromJql(searchStr, userInput);
    };

    fetchEpicsFromJql = async (jqlSearch: string, userInput: string): Promise<ReactSelectOption[]> => {
        const epicIssueType = this.props.issueMetadata.projects[0].issuetypes.find(isEpicIssueType);
        if (!epicIssueType) {
            return [];
        }

        const epicNameTypeId = Object.keys(epicIssueType.fields).find((key) => isEpicNameField(epicIssueType.fields[key]));
        if (!epicNameTypeId) {
            return [];
        }

        const projectKey = this.props.issueMetadata.projects[0].key;
        const fullJql = `project=${projectKey} and issuetype=${epicIssueType.id} ${jqlSearch} ${searchDefaults}`;

        const params = {
            jql: fullJql,
            fields: epicNameTypeId,
            q: userInput,
            instance_id: this.props.instanceID,
        };

        return this.props.searchIssues(params).then(({data}: {data: JiraIssue[]}) => {
            return data.map((issue) => ({
                value: issue.key,
                label: `${issue.key}: ${issue.fields[epicNameTypeId]}`,
            }));
        }).catch((e) => {
            this.setState({error: e});
            return [];
        });
    };

    render = (): JSX.Element => {
        return (
            <BackendSelector
                {...this.props}
                fetchInitialSelectedValues={this.fetchInitialSelectedValues}
                search={this.searchIssues}
            />
        );
    }
}
