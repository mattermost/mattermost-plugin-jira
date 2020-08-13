// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import {useFieldForIssueMetadata} from 'testdata/jira-issue-metadata-helpers';

import {FilterFieldInclusion} from 'types/model';
import {getCustomFieldFiltersForProjects} from 'utils/jira_issue_metadata';

import ChannelSettingsFilters, {Props} from './channel_settings_filters';

describe('components/ChannelSettingsFilters', () => {
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
        theme: {},
        fields: getCustomFieldFiltersForProjects(issueMetadata, [issueMetadata.projects[0].key]),
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
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<ChannelSettingsFilters>(
            <ChannelSettingsFilters {...props}/>
        );

        wrapper.setState({showCreateRow: true});
        expect(wrapper).toMatchSnapshot();
    });
});
