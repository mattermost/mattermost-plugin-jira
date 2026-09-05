// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render, screen} from '@testing-library/react';

import {RHSIssue} from 'types/rhs';

import RHSIssueRow from './rhs_issue_row';

const nowMs = Date.parse('2026-01-02T01:00:00Z');

const issueOne: RHSIssue = {
    key: 'TES-1',
    summary: 'One',
    browseUrl: 'https://example.atlassian.net/browse/TES-1',
    status: {name: 'In Progress', categoryKey: 'indeterminate'},
    priority: 'Medium',
    issueType: 'Task',
    project: 'TES',
    assignee: 'alice',
    reporter: 'bob',
    created: '2026-01-01T00:00:00Z',
    updated: '2026-01-02T00:00:00Z',
    dueDate: '',
    labels: [],
};

describe('components/rhs/rhs_issue_row', () => {
    test('row has one browse link and keeps the summary as text', () => {
        const {container} = render(
            <RHSIssueRow
                issue={issueOne}
                nowMs={nowMs}
            />,
        );

        const links = screen.getAllByRole('link');
        expect(links).toHaveLength(1);
        expect(links[0]).toHaveAttribute('href', 'https://example.atlassian.net/browse/TES-1');
        expect(links[0]).toHaveTextContent('TES-1');
        expect(links[0]).toHaveTextContent('One');
        expect(screen.getByText('One')).toHaveClass('jira-rhs-issue-summary');
        expect(container.innerHTML).not.toContain('api.atlassian.com');
    });

    test('long summary keeps full text and uses the clamp class', () => {
        const summary = 'x'.repeat(400);
        render(
            <RHSIssueRow
                issue={{...issueOne, summary}}
                nowMs={nowMs}
            />,
        );

        const summaryEl = screen.getByTitle(summary);
        expect(summaryEl).toHaveClass('jira-rhs-issue-summary');
        expect(summaryEl.textContent).toHaveLength(400);
    });

    test('status pill class follows statusCategory key', () => {
        render(
            <RHSIssueRow
                issue={{
                    ...issueOne,
                    status: {name: 'Done', categoryKey: 'done'},
                }}
                nowMs={nowMs}
            />,
        );

        expect(screen.getByText('Done')).toHaveClass('jira-rhs-status--done');
    });
});
