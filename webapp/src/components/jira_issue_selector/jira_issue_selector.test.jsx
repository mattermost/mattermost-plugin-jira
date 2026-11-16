// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import JiraIssueSelector from './jira_issue_selector';

describe('components/JiraIssueSelector', () => {
    const issues = [
        {key: 'KT-1', fields: {summary: 'Login fails'}},
        {key: 'KT-2', fields: {summary: 'Logout hangs'}},
    ];

    const renderSelector = (props = {}) => {
        const baseProps = {
            instanceID: 'https://something.atlassian.net',
            searchIssues: jest.fn().mockResolvedValue({data: issues}),
            onChange: jest.fn(),
            addValidate: jest.fn(),
            removeValidate: jest.fn(),
            ...props,
        };
        render(<JiraIssueSelector {...baseProps}/>);
        return baseProps;
    };

    test('should load issues on focus and emit the picked issue key', async () => {
        const props = renderSelector();

        await userEvent.click(screen.getByRole('combobox', {name: 'Jira Issue'}));
        await userEvent.click(await screen.findByRole('option', {name: 'KT-2: Logout hangs'}));

        expect(props.searchIssues).toHaveBeenCalledWith({fields: 'key,summary', q: '', instance_id: 'https://something.atlassian.net'});
        expect(props.onChange).toHaveBeenCalledWith('KT-2');
    });

    test('should search with the typed text', async () => {
        const props = renderSelector();

        await userEvent.type(screen.getByRole('combobox', {name: 'Jira Issue'}), ' login ');

        await waitFor(() => expect(props.searchIssues).toHaveBeenCalledWith(expect.objectContaining({q: 'login'})));
    });

    test('should show the search error', async () => {
        renderSelector({searchIssues: jest.fn().mockRejectedValue(new Error('Jira is down'))});

        await userEvent.click(screen.getByRole('combobox', {name: 'Jira Issue'}));

        expect(await screen.findByText('Error: Jira is down')).toBeInTheDocument();
    });
});
