// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ReactSelectOption} from 'types/model';
import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';
import {TEAM_FIELD} from 'constant';

const stripHTML = (text: string): string => {
    if (!text) {
        return text;
    }
    const doc = new DOMParser().parseFromString(text, 'text/html');
    return doc.body.textContent || '';
};

type Props = Omit<BackendSelectorProps, 'fetchInitialSelectedValues' | 'search'> & {
    fieldName: string;
    instanceID: string;
    searchTeamFields: (params: { fieldValue: string; instance_id: string }) => Promise<{
        data: { items: { Name: string; ID: string }[] };
        error?: Error;
    }>;
};

const JiraTeamSelector = (props: Props): JSX.Element => {
    const {value, instanceID, searchTeamFields} = props;

    const teamFields = async (inputValue: string): Promise<ReactSelectOption[]> => {
        const params = {
            fieldValue: inputValue,
            instance_id: instanceID,
        };

        return searchTeamFields(params).then(({data}) => {
            if (!data || !Array.isArray(data)) {
                return [];
            }

            return data.map((team) => ({
                value: team.ID,
                label: stripHTML(team.Name),
            }));
        });
    };

    const fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        const all = await teamFields('');
        if (!value) {
            return [];
        }

        return all.filter((option) => option.value === value);
    };

    return (
        <BackendSelector
            {...props}
            isMulti={false}
            fetchInitialSelectedValues={fetchInitialSelectedValues}
            search={teamFields}
            fieldKey={TEAM_FIELD}
        />
    );
};

export default JiraTeamSelector;
