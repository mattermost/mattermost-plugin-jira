// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {useDispatch} from 'react-redux';

import {getProjectRoles} from 'actions';

import {ReactSelectOption} from 'types/model';

import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';

const stripHTML = (text: string) => {
    if (!text) {
        return text;
    }

    const doc = new DOMParser().parseFromString(text, 'text/html');
    return doc.body.textContent || '';
};

type Props = BackendSelectorProps & {
    searchAutoCompleteFields: (params: {fieldValue: string; fieldName: string}) => (
        Promise<{data: {results: {value: string; displayName: string}[]}; error?: Error}>
    );
    fieldName: string;
};

export default function JiraRoleSelector(props: Props) {
    const [roles, setRoles] = React.useState([]);
    const dispatch = useDispatch();

    const projectKey = props.issueMetadata.projects[0].key;

    const fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        const roles = await dispatch(getProjectRoles({
            project: projectKey,
            instance_id: props.instanceID,
        }));

        setRoles(roles);
        return roles;
    };

    const searchAutoCompleteFields = async (inputValue: string): Promise<ReactSelectOption[]> => {
        return roles;
    };

    render = (): JSX.Element => {
        return (
            <BackendSelector
                {...props}
                fetchInitialSelectedValues={fetchInitialSelectedValues}
                search={searchAutoCompleteFields}
            />
        );
    };
}
