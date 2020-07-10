// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import Preferences from 'mattermost-redux/constants/preferences';

import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';

import {IssueMetadata} from 'types/model';

import JiraEpicSelector from './jira_epic_selector';

describe('components/JiraEpicSelector', () => {
    const baseProps = {
        searchIssues: jest.fn().mockResolvedValue({}),
        issueMetadata: issueMetadata as IssueMetadata,
        theme: Preferences.THEMES.default,
        isMulti: true,
        onChange: jest.fn(),
        value: ['KT-17', 'KT-20'],
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<JiraEpicSelector>(
            <JiraEpicSelector {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('#searchIssues should call searchIssues', () => {
        const searchIssues = jest.fn().mockResolvedValue({});

        const props = {
            ...baseProps,
            searchIssues,
        };
        const wrapper = shallow<JiraEpicSelector>(
            <JiraEpicSelector {...props}/>
        );

        wrapper.instance().searchIssues('');

        let args = props.searchIssues.mock.calls[0][0];
        expect(args).toEqual({
            fields: 'customfield_10011',
            jql: 'project=KT and issuetype=10000  ORDER BY updated DESC',
            q: '',
        });

        wrapper.instance().searchIssues('some input');

        args = props.searchIssues.mock.calls[1][0];
        expect(args).toEqual({
            fields: 'customfield_10011',
            jql: 'project=KT and issuetype=10000  and ("Epic Name"~"some input" or "Epic Name"~"some input*") ORDER BY updated DESC',
            q: 'some input',
        });
    });
});
