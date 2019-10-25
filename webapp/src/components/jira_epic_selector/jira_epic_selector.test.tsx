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
        fetchEpicsWithParams: jest.fn(),
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

    test('should call fetchEpicsWithParams on mount', () => {
        const props = {...baseProps};
        const wrapper = shallow<JiraEpicSelector>(
            <JiraEpicSelector {...props}/>
        );
        expect(props.fetchEpicsWithParams).toHaveBeenCalledWith({
            epic_name_type_id: 'customfield_10011',
            jql: 'project=KT and issuetype=10000 and id IN (KT-17, KT-20) ORDER BY updated DESC',
            q: '',
        });
    });

    test('#searchIssues should call fetchEpicsWithParams', () => {
        const props = {...baseProps};
        const wrapper = shallow<JiraEpicSelector>(
            <JiraEpicSelector {...props}/>
        );

        wrapper.instance().searchIssues('some input');

        expect(props.fetchEpicsWithParams).toHaveBeenCalledWith({
            epic_name_type_id: 'customfield_10011',
            jql: 'project=KT and issuetype=10000 and "Epic Name"~"some%20input*" ORDER BY updated DESC',
            q: 'some input',
        });
    });
});
