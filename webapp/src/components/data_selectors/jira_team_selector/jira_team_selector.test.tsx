// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, screen} from '@testing-library/react';

import Preferences from 'mattermost-redux/constants/preferences';

import {IssueMetadata, TeamItem} from 'types/model';
import {renderWithRedux} from 'testlib/test-utils';

import JiraTeamSelector from './jira_team_selector';

describe('components/JiraTeamSelector', () => {
    // Keys match the server's TeamList json tags. Changing them on either side
    // silently blanks out every option, as in MM-70879.
    const teams: TeamItem[] = [
        {id: 'team-1', name: 'Alpha Team'},
        {id: 'team-2', name: '<b>Beta</b> Team'},
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

    test('should not render teams missing an id or a name', async () => {
        const searchTeamFields = jest.fn().mockResolvedValue({
            data: [{id: 'team-1'}, {name: 'Alpha Team'}],
        });

        await renderSelector({value: 'team-1', searchTeamFields});

        expect(screen.queryByText('Alpha Team')).toBeNull();
    });
});
