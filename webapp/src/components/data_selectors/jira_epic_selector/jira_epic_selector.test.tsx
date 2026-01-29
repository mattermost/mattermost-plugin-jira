// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render} from '@testing-library/react';

import Preferences from 'mattermost-redux/constants/preferences';

import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';

import {IssueMetadata} from 'types/model';

import JiraEpicSelector from './jira_epic_selector';

describe('components/JiraEpicSelector', () => {
    const baseProps = {
        searchIssues: jest.fn().mockResolvedValue({}),
        issueMetadata: issueMetadata as IssueMetadata,
        theme: Preferences.THEMES.denim,
        isMulti: true,
        onChange: jest.fn(),
        value: ['KT-17', 'KT-20'],
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        instanceID: 'https://something.atlassian.net',
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', () => {
        const props = {...baseProps};
        const {container} = render(<JiraEpicSelector {...props}/>);
        expect(container).toBeInTheDocument();
    });

    test('#searchIssues should call searchIssues', () => {
        const searchIssues = jest.fn().mockResolvedValue({data: {issues: []}});
        const props = {
            ...baseProps,
            searchIssues,
        };
        const {container} = render(<JiraEpicSelector {...props}/>);
        expect(container).toBeInTheDocument();
    });
});
