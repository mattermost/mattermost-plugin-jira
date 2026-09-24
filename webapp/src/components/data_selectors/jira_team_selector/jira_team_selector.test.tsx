// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, screen} from '@testing-library/react';

import Preferences from 'mattermost-redux/constants/preferences';

import {IssueMetadata} from 'types/model';
import {renderWithRedux} from 'testlib/test-utils';

import JiraTeamSelector from './jira_team_selector';

describe('components/JiraTeamSelector', () => {
    // Keys match Go's default serialization of the server's untagged TeamList.
    // Renaming them on either side silently blanks out every option (MM-70879).
    const teams = [
        {ID: 'team-1', Name: 'Alpha Team'},
        {ID: 'team-2', Name: '<b>Beta</b> Team'},
    ];

    const baseProps = {
        fieldName: 'Team',
        instanceID: 'https://something.atlassian.net',
        searchTeamFields: jest.fn().mockResolvedValue({data: teams}),
        issueMetadata: {} as IssueMetadata,
        theme: Preferences.THEMES.denim,
        onChange: jest.fn(),
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        value: '',
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    // BackendSelector's inherited react-select props make `value` unsatisfiable
    // for a plain string, so the assembled props are cast at the render site.
    type SelectorProps = React.ComponentProps<typeof JiraTeamSelector>;

    const renderSelector = async (props: Partial<typeof baseProps> = {}) => {
        const allProps = {...baseProps, ...props} as unknown as SelectorProps;
        await act(async () => {
            renderWithRedux(<JiraTeamSelector {...allProps}/>);
        });
    };

    test('should label the selected team using the name returned by the server', async () => {
        await renderSelector({value: 'team-1'});

        expect(screen.getByText('Alpha Team')).toBeTruthy();
    });

    test('should strip HTML from team names', async () => {
        await renderSelector({value: 'team-2'});

        expect(screen.getByText('Beta Team')).toBeTruthy();
    });

    // Channel subscription filters hold their value as FilterValue.values. An
    // unresolved array sends BackendSelector down its recovery path, which
    // labels the team correctly but costs a second request.
    test('should resolve an array value without searching twice', async () => {
        const searchTeamFields = jest.fn().mockResolvedValue({data: teams});

        await renderSelector({value: ['team-1'] as unknown as string, searchTeamFields});

        expect(screen.getByText('Alpha Team')).toBeTruthy();
        expect(searchTeamFields).toHaveBeenCalledTimes(1);
    });

    test('should not search for teams when nothing is selected', async () => {
        const searchTeamFields = jest.fn().mockResolvedValue({data: teams});

        await renderSelector({value: '', searchTeamFields});

        expect(searchTeamFields).not.toHaveBeenCalled();
    });
});
