// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import Preferences from 'mattermost-redux/constants/preferences';

import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';

import {IssueMetadata} from 'types/model';

import JiraEpicSelector from './jira_epic_selector';

const mockStore = configureStore([thunk]);

const defaultMockState = {
    'plugins-jira': {
        installedInstances: [],
        connectedInstances: [],
    },
    entities: {
        general: {
            config: {
                SiteURL: 'http://localhost:8065',
            },
        },
    },
};

const renderWithRedux = (ui: React.ReactElement, initialState = defaultMockState) => {
    const store = mockStore(initialState);
    return {
        store,
        ...render(
            <IntlProvider locale='en'>
                <Provider store={store}>{ui}</Provider>
            </IntlProvider>,
        ),
    };
};

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

    test('should render component', async () => {
        const props = {...baseProps};
        const ref = React.createRef<JiraEpicSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraEpicSelector
                    {...props}
                    ref={ref}
                />,
            );
        });

        expect(ref.current).toBeDefined();
    });

    test('#searchIssues should call searchIssues', async () => {
        const searchIssues = jest.fn().mockResolvedValue({data: []});

        const props = {
            ...baseProps,
            searchIssues,
        };
        const ref = React.createRef<JiraEpicSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraEpicSelector
                    {...props}
                    ref={ref}
                />,
            );
        });

        searchIssues.mockClear();

        await act(async () => {
            await ref.current?.searchIssues('');
        });

        let args = searchIssues.mock.calls[0][0];
        expect(args).toEqual({
            fields: 'customfield_10011',
            jql: 'project=KT and issuetype=10000  ORDER BY updated DESC',
            q: '',
            instance_id: 'https://something.atlassian.net',
        });

        await act(async () => {
            await ref.current?.searchIssues('some input');
        });

        args = searchIssues.mock.calls[1][0];
        expect(args).toEqual({
            fields: 'customfield_10011',
            jql: 'project=KT and issuetype=10000  and ("Epic Name"~"some input" or "Epic Name"~"some input*") ORDER BY updated DESC',
            q: 'some input',
            instance_id: 'https://something.atlassian.net',
        });
    });
});
