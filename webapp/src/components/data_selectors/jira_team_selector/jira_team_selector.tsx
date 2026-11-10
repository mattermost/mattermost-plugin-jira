// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ReactSelectOption} from 'types/model';
import BackendSelector, {Props as BackendSelectorProps} from '../backend_selector';
import {TEAM_FIELD} from '../../../constant';

const stripHTML = (text: string): string => {
    if (!text) {
        return text;
    }
    const doc = new DOMParser().parseFromString(text, 'text/html');
    return doc.body.textContent || '';
};

type TeamItem = {name: string; id: string};

type Props = Omit<BackendSelectorProps, 'fetchInitialSelectedValues' | 'search' | 'value'> & {
    fieldName: string;
    instanceID: string;
    value?: string;
    searchTeamFields: (params: {fieldValue: string; instance_id: string}) => Promise<{data: TeamItem[]}>;
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

        return searchTeamFields(params).then(({data}: {data: TeamItem[]}) => {
            if (!data || !Array.isArray(data)) {
                return [];
            }

            return data.map((team: TeamItem) => ({
                value: team.id,
                label: stripHTML(team.name),
            }));
        });
    };

    const fetchInitialSelectedValues = async (): Promise<ReactSelectOption[]> => {
        if (!value) {
            return [];
        }

        const all = await teamFields('');
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
