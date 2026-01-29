// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import {useFieldForIssueMetadata} from 'testdata/jira-issue-metadata-helpers';

import {FilterFieldInclusion} from 'types/model';
import {getCustomFieldFiltersForProjects} from 'utils/jira_issue_metadata';

import ChannelSubscriptionFilters, {Props} from './channel_subscription_filters';

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

describe('components/ChannelSubscriptionFilters', () => {
    const field = {
        hasDefaultValue: false,
        key: 'priority',
        name: 'Priority',
        operations: [
            'set',
        ],
        required: false,
        schema: {
            system: 'priority',
            type: 'priority',
        },
        allowedValues: [{
            id: '1',
            name: 'Highest',
        }, {
            id: '2',
            name: 'High',
        }, {
            id: '3',
            name: 'Medium',
        }, {
            id: '4',
            name: 'Low',
        }, {
            id: '5',
            name: 'Lowest',
        }],
    };

    const issueMetadata = useFieldForIssueMetadata(field, 'priority');

    const mockTheme = {
        centerChannelColor: '#333333',
        centerChannelBg: '#ffffff',
        buttonBg: '#166de0',
        buttonColor: '#ffffff',
        linkColor: '#2389d7',
        errorTextColor: '#fd5960',
    };

    const baseProps: Props = {
        theme: mockTheme,
        fields: getCustomFieldFiltersForProjects(issueMetadata, [issueMetadata.projects[0].key], []),
        values: [{
            key: 'priority',
            inclusion: FilterFieldInclusion.INCLUDE_ANY,
            values: ['1'],
        }],
        chosenIssueTypes: ['10001'],
        issueMetadata,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        onChange: jest.fn(),
        instanceID: 'https://something.atlassian.net',
        securityLevelEmptyForJiraSubscriptions: true,
        searchTeamFields: jest.fn().mockResolvedValue({data: []}),
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', async () => {
        const props = {...baseProps};
        const ref = React.createRef<ChannelSubscriptionFilters>();
        await act(async () => {
            renderWithRedux(
                <ChannelSubscriptionFilters
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            ref.current?.setState({showCreateRow: true});
        });

        expect(ref.current).toBeDefined();
    });
});
