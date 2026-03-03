// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act} from '@testing-library/react';

import Preferences from 'mattermost-redux/constants/preferences';
import {Team} from '@mattermost/types/teams';

import projectMetadata from 'testdata/cloud-get-jira-project-metadata.json';
import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project.json';

import {IssueMetadata} from 'types/model';
import {renderWithRedux} from 'testlib/test-utils';

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
        theme: Preferences.THEMES.denim,
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

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should render component', async () => {
        const props = {...baseProps};
        const ref = React.createRef<CreateIssueForm>();
        await act(async () => {
            renderWithRedux(
                <CreateIssueForm
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });
        expect(ref.current).toBeDefined();
    });

    test('should render component with no issue metadata', async () => {
        const props = {...baseProps};
        const ref = React.createRef<CreateIssueForm>();
        await act(async () => {
            renderWithRedux(
                <CreateIssueForm
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState({...baseState, jiraIssueMetadata: null});
        });
        expect(ref.current).toBeDefined();
    });

    test('should render component with no instance id', async () => {
        const props = {...baseProps};
        const ref = React.createRef<CreateIssueForm>();
        await act(async () => {
            renderWithRedux(
                <CreateIssueForm
                    {...props}
                    ref={ref}
                />,
            );
        });
        expect(ref.current).toBeDefined();
    });

    test('should render component with form filled', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const ref = React.createRef<CreateIssueForm>();
        await act(async () => {
            renderWithRedux(
                <CreateIssueForm
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });
        const fields = ref.current?.state.fields;

        await act(async () => {
            ref.current?.setState({
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
        });
        expect(ref.current).toBeDefined();
    });

    test('should render component with error', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const ref = React.createRef<CreateIssueForm>();
        await act(async () => {
            renderWithRedux(
                <CreateIssueForm
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });
        const fields = ref.current?.state.fields;

        await act(async () => {
            ref.current?.setState({
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
        });

        await act(async () => {
            ref.current?.setState({instanceID: 'https://something.atlassian.net'});
        });
        expect(ref.current).toBeDefined();
    });

    test('should call create prop to create an issue', async () => {
        const create = jest.fn().mockResolvedValue({});
        const props = {...baseProps, create};
        const ref = React.createRef<CreateIssueForm>();
        await act(async () => {
            renderWithRedux(
                <CreateIssueForm
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });
        const fields = ref.current?.state.fields;

        await act(async () => {
            ref.current?.setState({
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
        });

        if (ref.current) {
            ref.current.validator = {validate: () => true, addComponent: jest.fn(), removeComponent: jest.fn()};
        }

        await act(async () => {
            ref.current?.handleSubmit();
        });
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
        const ref = React.createRef<CreateIssueForm>();
        await act(async () => {
            renderWithRedux(
                <CreateIssueForm
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });
        const fields = ref.current?.state.fields;

        await act(async () => {
            ref.current?.setState({
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
        });

        if (ref.current) {
            ref.current.validator = {validate: () => true, addComponent: jest.fn(), removeComponent: jest.fn()};
        }

        await act(async () => {
            ref.current?.handleSubmit();
        });
        expect(create).toHaveBeenCalled();
    });
});
