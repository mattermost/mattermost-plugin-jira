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
    }>;
    getSprintByID: (params: {instance_id: string; sprint_id: string}) => Promise<{
        data: Sprint;
    }>;
};

const JiraSprintSelector = (props: Props): JSX.Element => {
    const {value, instanceID, projectKey, searchSprints, getSprintByID} = props;

    const fetchSprints = async (query: string): Promise<ReactSelectOption[]> => {
        if (!instanceID || !projectKey) {
            return [];
        }

        const params = {
            instance_id: instanceID,
            project_key: projectKey,
        };

        try {
            const {data} = await searchSprints(params);
            if (!data || !Array.isArray(data)) {
                return [];
            }

            const options = data.map((sprint: Sprint) => ({
                value: String(sprint.id),
                label: `${sprint.name} (${sprint.state})`,
            }));

            if (!query) {
                return options;
            }

            const lowerQuery = query.toLowerCase();
            return options.filter((opt) => opt.label.toLowerCase().includes(lowerQuery));
        } catch {
            return [];
        }
    };

    const fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        if (!value || !instanceID) {
            return [];
        }

        const sprintID = String(value);
        try {
            const {data} = await getSprintByID({
                instance_id: instanceID,
                sprint_id: sprintID,
            });
            if (!data) {
                return [];
            }
            return [{
                value: String(data.id),
                label: `${data.name} (${data.state})`,
            }];
        } catch {
            return [];
        }
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
