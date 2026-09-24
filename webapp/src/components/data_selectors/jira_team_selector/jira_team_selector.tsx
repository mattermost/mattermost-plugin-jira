// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ReactSelectOption, SearchTeamFields, TeamItem} from 'types/model';
import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';
import {TEAM_FIELD} from '../../../constant';

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
    searchTeamFields: SearchTeamFields;
};

const JiraTeamSelector = (props: Props): JSX.Element => {
    const {value, instanceID, searchTeamFields} = props;

    const teamFields = async (inputValue: string): Promise<ReactSelectOption[]> => {
        if (!instanceID) {
            return [];
        }

        const params = {
            fieldValue: inputValue,
            instance_id: instanceID,
        };

        return searchTeamFields(params).then(({data}) => {
            if (!data || !Array.isArray(data)) {
                return [];
            }

            // Drop entries the server could not fully populate, so they never
            // render as blank options.
            const teams = data.filter((team: TeamItem) => team && team.id && team.name);

            return teams.map((team: TeamItem) => ({
                value: team.id,
                label: stripHTML(team.name),
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
