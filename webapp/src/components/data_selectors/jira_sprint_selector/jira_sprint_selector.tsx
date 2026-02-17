// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ReactSelectOption} from 'types/model';
import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';

type Sprint = {
    id: number;
    name: string;
    state: string;
};

type Props = Omit<BackendSelectorProps, 'fetchInitialSelectedValues' | 'search'> & {
    instanceID: string;
    projectKey: string;
    searchSprints: (params: {instance_id: string; project_key: string}) => Promise<{
        data: Sprint[];
        error?: Error;
    }>;
};

const JiraSprintSelector = (props: Props): JSX.Element => {
    const {value, instanceID, projectKey, searchSprints} = props;

    const fetchSprints = async (): Promise<ReactSelectOption[]> => {
        if (!instanceID || !projectKey) {
            return [];
        }

        const params = {
            instance_id: instanceID,
            project_key: projectKey,
        };

        try {
            const {data, error} = await searchSprints(params);
            if (error) {
                console.warn('Failed to fetch sprints:', error);
                return [];
            }
            if (!data || !Array.isArray(data)) {
                return [];
            }

            return data.map((sprint: Sprint) => ({
                value: String(sprint.id),
                label: `${sprint.name} (${sprint.state})`,
            }));
        } catch (e) {
            console.warn('Failed to fetch sprints:', e);
            return [];
        }
    };

    const fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        const all = await fetchSprints();
        if (!value) {
            return [];
        }

        const valueStr = String(value);
        return all.filter((option) => option.value === valueStr);
    };

    return (
        <BackendSelector
            {...props}
            isMulti={false}
            fetchInitialSelectedValues={fetchInitialSelectedValues}
            search={fetchSprints}
        />
    );
};

export default JiraSprintSelector;
