// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render} from '@testing-library/react';

import RHSIssueTypeIcon, {rhsIssueTypeIconName} from './rhs_issue_type_icon';

describe('components/rhs/rhs_issue_type_icon', () => {
    test('rhsIssueTypeIconName maps Bug to bug', () => {
        expect(rhsIssueTypeIconName('Bug')).toBe('bug');
    });

    test('rhsIssueTypeIconName maps Sub-task and custom subtask names to subtask', () => {
        expect(rhsIssueTypeIconName('Sub-task')).toBe('subtask');
        expect(rhsIssueTypeIconName('Custom Subtask')).toBe('subtask');
    });

    test('rhsIssueTypeIconName maps an unknown type to task', () => {
        expect(rhsIssueTypeIconName('Spike')).toBe('task');
    });

    test('renders a bug class for Bug', () => {
        const {container} = render(
            <RHSIssueTypeIcon
                issueType='Bug'
            />,
        );
        expect(container.querySelector('.jira-rhs-type-icon--bug')).not.toBeNull();
    });

    test('renders Jira type icon URL when provided', () => {
        const {container} = render(
            <RHSIssueTypeIcon
                issueType='Bug'
                iconUrl='https://example.atlassian.net/bug.svg'
            />,
        );
        const img = container.querySelector('img');
        expect(img).not.toBeNull();
        expect(img).toHaveAttribute('src', 'https://example.atlassian.net/bug.svg');
        expect(img).toHaveAttribute('alt', 'Bug');
        expect(container.querySelector('svg')).toBeNull();
    });
});
