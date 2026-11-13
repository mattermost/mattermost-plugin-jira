// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import BackendSelector from 'components/data_selectors/backend_selector';

const fetchNoInitialValues = () => Promise.resolve([]);

export default function JiraIssueSelector(props) {
    const searchIssues = async (text) => {
        const params = {
            fields: 'key,summary',
            q: text.trim(),
            instance_id: props.instanceID,
        };

        const {data} = await props.searchIssues(params);
        if (!data) {
            return [];
        }
        return data.map((issue) => ({
            value: issue.key,
            label: `${issue.key}: ${issue.fields.summary}`,
        }));
    };

    return (
        <BackendSelector
            name='issue'
            label='Jira Issue'
            placeholder='Search for issues containing text...'
            instanceID={props.instanceID}
            value={props.value}
            onChange={props.onChange}
            required={props.required}
            isClearable={true}
            addValidate={props.addValidate}
            removeValidate={props.removeValidate}
            fetchInitialSelectedValues={fetchNoInitialValues}
            search={searchIssues}
        />
    );
}

JiraIssueSelector.propTypes = {
    required: PropTypes.bool,
    onChange: PropTypes.func.isRequired,
    searchIssues: PropTypes.func.isRequired,
    value: PropTypes.string,
    addValidate: PropTypes.func.isRequired,
    removeValidate: PropTypes.func.isRequired,
    instanceID: PropTypes.string.isRequired,
};
