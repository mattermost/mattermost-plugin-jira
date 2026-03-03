// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act} from '@testing-library/react';

import {useFieldForIssueMetadata} from 'testdata/jira-issue-metadata-helpers';

import {FilterFieldInclusion} from 'types/model';
import {getCustomFieldFiltersForProjects} from 'utils/jira_issue_metadata';
import {mockTheme, renderWithRedux} from 'testlib/test-utils';

import ChannelSubscriptionFilters, {Props} from './channel_subscription_filters';

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

    test('should render component', async () => {
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
