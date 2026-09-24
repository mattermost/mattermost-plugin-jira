// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {fireEvent, render, screen} from '@testing-library/react';

import Preferences from 'mattermost-redux/constants/preferences';

import JiraField from './jira_field';

// The selector is connected to redux and fetches on mount, so it is replaced
// with a stub that exposes the value it receives and the value it emits.
jest.mock('./data_selectors/jira_team_selector', () => ({
    __esModule: true,
    default: (props) => (
        <div>
            <span data-testid='selector-value'>{String(props.value)}</span>
            <button onClick={() => props.onChange('team-1')}>{'select'}</button>
            <button onClick={() => props.onChange('')}>{'clear'}</button>
        </div>
    ),
}));

describe('components/JiraField team field', () => {
    // The server reads the submitted Team field as m["id"], and create-meta
    // ships teams as {id, name} — both lowercase, unlike the capitalized
    // get-team-fields payload. Renaming this key breaks issue creation.
    const teamField = {
        key: 'customfield_10800',
        name: 'Team',
        required: false,
        schema: {
            type: 'any',
            custom: 'com.atlassian.jira.plugin.system.customfieldtypes:atlassian-team',
        },
    };

    const baseProps = {
        id: 'customfield_10800',
        instanceID: 'https://something.atlassian.net',
        field: teamField,
        projectKey: 'KT',
        issueMetadata: {projects: []},
        theme: Preferences.THEMES.denim,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    const renderField = (props = {}) => {
        const onChange = jest.fn();
        render(
            <JiraField
                {...baseProps}
                onChange={onChange}
                {...props}
            />,
        );

        return onChange;
    };

    test('should submit the selected team as an object keyed by id', () => {
        const onChange = renderField();

        fireEvent.click(screen.getByText('select'));

        expect(onChange).toHaveBeenCalledWith('customfield_10800', {id: 'team-1'});
    });

    test('should clear the team field when the selection is removed', () => {
        const onChange = renderField();

        fireEvent.click(screen.getByText('clear'));

        expect(onChange).toHaveBeenCalledWith('customfield_10800', null);
    });

    test('should unwrap a saved team value back into the selector', () => {
        renderField({value: {id: 'team-1'}});

        expect(screen.getByTestId('selector-value').textContent).toEqual('team-1');
    });
});
