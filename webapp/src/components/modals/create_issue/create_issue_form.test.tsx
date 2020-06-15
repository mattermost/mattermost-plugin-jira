// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow, mount} from 'enzyme';

import Preferences from 'mattermost-redux/constants/preferences';
import {Team} from 'mattermost-redux/types/teams';

import projectMetadata from 'testdata/cloud-get-jira-project-metadata.json';
import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project.json';

import {IssueMetadata} from 'types/model';

import CreateIssueForm from './create_issue_form';

describe('components/CreateIssue', () => {
    const baseActions = {
        clearIssueMetadata: jest.fn().mockResolvedValue({}),
        fetchJiraIssueMetadataForProjects: jest.fn().mockResolvedValue({}),
        fetchJiraProjectMetadata: jest.fn().mockResolvedValue({}),
        create: jest.fn().mockResolvedValue({}),
    };

    const baseProps = {
        ...baseActions,
        theme: Preferences.THEMES.default,
        jiraProjectMetadata: projectMetadata,
        jiraIssueMetadata: issueMetadata,
        currentTeam: {name: 'Team1'} as Team,
        close: jest.fn(),
        visible: true,
        channelId: 'channel-id-1',
    };

    const baseState = {
        instanceID: 'https://something.atlassian.net',
        jiraIssueMetadata: issueMetadata as IssueMetadata,
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<CreateIssueForm>(
            <CreateIssueForm {...props}/>
        );
        wrapper.setState(baseState);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no issue metadata', () => {
        const props = {...baseProps};
        const wrapper = shallow<CreateIssueForm>(
            <CreateIssueForm {...props}/>
        );
        wrapper.setState({...baseState, jiraIssueMetadata: null});
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no instance id', () => {
        const props = {...baseProps};
        const wrapper = shallow<CreateIssueForm>(
            <CreateIssueForm {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with form filled', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const wrapper = shallow<CreateIssueForm>(
            <CreateIssueForm {...props}/>
        );
        wrapper.setState(baseState);
        const fields = wrapper.state('fields');

        wrapper.setState({
            fields: {
                ...fields,
                summary: '',
                description: 'some description',
                project: {key: 'KT'},
                issuetype: {id: '10001'},
                priority: {id: 1},
            },
            projectKey: 'KT',
            issueType: '10001',
        });
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with error', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const wrapper = shallow<CreateIssueForm>(
            <CreateIssueForm {...props}/>
        );
        wrapper.setState(baseState);
        const fields = wrapper.state('fields');

        wrapper.setState({
            fields: {
                ...fields,
                summary: '',
                description: 'some description',
                project: {key: 'KT'},
                issuetype: {id: '10001'},
                priority: {id: 1},
            },
            projectKey: 'KT',
            issueType: '10001',
            error: 'Some error',
        });

        wrapper.setState({instanceID: 'https://something.atlassian.net'});
        expect(wrapper).toMatchSnapshot();
    });

    test('should call create prop to create an issue', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const wrapper = shallow<CreateIssueForm>(
            <CreateIssueForm {...props}/>
        );
        wrapper.setState(baseState);
        const fields = wrapper.state('fields');

        wrapper.setState({
            fields: {
                ...fields,
                summary: 'some summary',
                description: 'some description',
                project: {key: 'KT'},
                issuetype: {id: '10001'},
                priority: {id: '1'},
            },
            projectKey: 'KT',
            issueType: '10001',
        });

        wrapper.instance().handleSubmit();
        expect(create).toHaveBeenCalled();
    });

    test('SERVER - should call create prop to create an issue', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            create,
            jiraProjectMetadata: serverProjectMetadata,
            jiraIssueMetadata: serverIssueMetadata,
        };
        const wrapper = shallow<CreateIssueForm>(
            <CreateIssueForm {...props}/>
        );
        wrapper.setState(baseState);
        const fields = wrapper.state('fields');

        wrapper.setState({
            fields: {
                ...fields,
                summary: 'some summary',
                description: 'some description',
                project: {key: 'HEY'},
                issuetype: {id: '10001'},
                priority: {id: '1'},
            },
            projectKey: 'HEY',
            issueType: '10001',
        });

        wrapper.instance().handleSubmit();
        expect(create).toHaveBeenCalled();
    });
});
