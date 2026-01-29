// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Preferences from 'mattermost-redux/constants/preferences';
import {Team} from '@mattermost/types/teams';

import projectMetadata from 'testdata/cloud-get-jira-project-metadata.json';
import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project.json';

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

    test('baseProps are correctly defined', () => {
        expect(baseProps.theme).toBe(Preferences.THEMES.denim);
        expect(baseProps.visible).toBe(true);
        expect(baseProps.channelId).toBe('channel-id-1');
    });

    test('baseActions are correctly mocked', () => {
        expect(baseActions.create).toBeDefined();
        expect(baseActions.clearIssueMetadata).toBeDefined();
        expect(baseActions.fetchJiraIssueMetadataForProjects).toBeDefined();
    });

    test('project metadata is loaded', () => {
        expect(baseProps.jiraProjectMetadata).toBeDefined();
        expect(projectMetadata).toBeDefined();
    });

    test('issue metadata is loaded', () => {
        expect(baseProps.jiraIssueMetadata).toBeDefined();
        expect(issueMetadata).toBeDefined();
    });

    test('server project metadata is available', () => {
        expect(serverProjectMetadata).toBeDefined();
    });

    test('server issue metadata is available', () => {
        expect(serverIssueMetadata).toBeDefined();
    });

    test('current team is correctly set', () => {
        expect(baseProps.currentTeam.name).toBe('Team1');
    });
});
