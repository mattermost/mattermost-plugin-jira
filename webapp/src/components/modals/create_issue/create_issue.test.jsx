// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow, mount} from 'enzyme';
import {Provider} from 'react-redux';
import configureStore from 'redux-mock-store';

import Preferences from 'mattermost-redux/constants/preferences';

import projectMetadata from 'testdata/cloud-get-jira-project-metadata.json';
import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project.json';

import CreateIssue from './create_issue';

const mockStore = configureStore();
const WrappedCreateIssue = (props) => {
    const store = mockStore({})
    return (
        <Provider store={store}>
            <CreateIssue {...props}/>
        </Provider>
    );
}

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
        currentTeam: {},
        close: jest.fn(),
        visible: true,
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow(
            <CreateIssue {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no issue metadata', () => {
        const props = {...baseProps, jiraIssueMetadata: null};
        const wrapper = shallow(
            <CreateIssue {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no project or issue metadata', () => {
        const props = {...baseProps, jiraIssueMetadata: null, jiraProjectMetadata: null};
        const wrapper = shallow(
            <CreateIssue {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with form filled', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const wrapper = shallow(
            <CreateIssue {...props}/>
        );
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
        const wrapper = shallow(
            <CreateIssue {...props}/>
        );
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

        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with getMetadataError', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const wrapper = shallow(
            <CreateIssue {...props}/>
        );
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
            getMetaDataError: 'Some error',
        });

        expect(wrapper).toMatchSnapshot();
    });

    test('should call create prop to create an issue', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const wrapper = mount(
            <WrappedCreateIssue {...props}/>
        ).find(CreateIssue);

        const fields = wrapper.find(CreateIssue).state('fields');

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

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(create).not.toHaveBeenCalled();

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

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
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
        const wrapper = mount(
            <CreateIssue {...props}/>
        );
        const fields = wrapper.state('fields');

        wrapper.setState({
            fields: {
                ...fields,
                summary: '',
                description: 'some description',
                project: {key: 'HEY'},
                issuetype: {id: '10001'},
                priority: {id: 1},
            },
            projectKey: 'HEY',
            issueType: '10001',
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(create).not.toHaveBeenCalled();

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

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(create).toHaveBeenCalled();
    });
});
