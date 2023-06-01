// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import JiraEpicSelector from 'components/data_selectors/jira_epic_selector';

import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';

import {FilterFieldInclusion, IssueMetadata, FilterField} from 'types/model';
import {getCustomFieldFiltersForProjects, isEpicLinkField} from 'utils/jira_issue_metadata';

import ChannelSubscriptionFilter, {Props} from './channel_subscription_filter';

describe('components/ChannelSubscriptionFilter', () => {
    const fields = getCustomFieldFiltersForProjects(issueMetadata, [issueMetadata.projects[0].key]);
    const baseProps: Props = {
        theme: {},
        fields,
        field: fields.find((f) => f.key === 'priority') as FilterField,
        value: {
            key: 'priority',
            inclusion: FilterFieldInclusion.INCLUDE_ANY,
            values: ['1'],
        },
        chosenIssueTypes: [],
        issueMetadata: issueMetadata as IssueMetadata,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        onChange: jest.fn(),
        removeFilter: jest.fn(),
        instanceID: 'https://something.atlassian.net',
        securityLevelEmptyForJiraSubscriptions: true,
    };

    test('should match snapshot', () => {
        const props = {...baseProps, issueMetadata: {}};
        const wrapper = shallow<ChannelSubscriptionFilter>(
            <ChannelSubscriptionFilter {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should render JiraEpicSelector when Epic Link field is selected', () => {
        const props = {...baseProps};
        const wrapper = shallow<ChannelSubscriptionFilter>(
            <ChannelSubscriptionFilter {...props}/>
        );

        expect(wrapper.find(JiraEpicSelector).length).toBe(0);

        wrapper.setProps({
            ...props,
            field: fields.find(isEpicLinkField) as FilterField,
        });

        expect(wrapper.find(JiraEpicSelector).length).toBe(1);
    });

    test('should render correct inclusion captions for different include choices', () => {
        const props = {...baseProps};

        const wrapper = shallow<ChannelSubscriptionFilter>(
            <ChannelSubscriptionFilter {...props}/>
        );

        const select = wrapper.find('ReactSelectSetting[name="inclusion"]');
        const func = select.props().formatOptionLabel;

        const tests = [
            ['include_any', 'Includes either of the values (or)'],
            ['include_all', 'Includes all of the values (and)'],
            ['exclude_any', 'Excludes all of the values'],
            ['empty', 'Includes when the value is empty'],
        ];

        // Select dropdown is open
        for (const t of tests) {
            const element = func({value: t[0]}, {});
            const wrapper2 = shallow(element);
            expect(wrapper2.text()).toEqual(t[1]);
        }

        // Select dropdown is closed
        const result = func({value: 'include_any', label: 'Some Option Label'}, {context: 'value'});
        expect(result).toEqual('Some Option Label');
    });

    test('checkFieldConflictError should return an error string when there is a conflict', () => {
        const props = {
            ...baseProps,
            chosenIssueTypes: ['10002'],
            field: {
                ...baseProps.field,
                issueTypes: [{id: '10002', name: 'Task'}],
            },
        };
        const wrapper = shallow<ChannelSubscriptionFilter>(
            <ChannelSubscriptionFilter {...props}/>
        );

        let result;
        result = wrapper.instance().checkFieldConflictError();
        expect(result).toBeNull();

        wrapper.setProps({
            ...props,
            chosenIssueTypes: ['10002'],
            field: {
                ...props.field,
                name: 'FieldName',
                issueTypes: [{id: '10003', name: 'Task'}],
            },
        });

        result = wrapper.instance().checkFieldConflictError();
        expect(result).toEqual('FieldName does not exist for issue type(s): Task.');
    });

    test('checkInclusionError should return an error string when there is an invalid inclusion value', () => {
        const props: Props = {
            ...baseProps,
            field: {
                ...baseProps.field,
                schema: {
                    ...baseProps.field.schema,
                    type: 'securitylevel',
                },
            },
        };
        const wrapper = shallow<ChannelSubscriptionFilter>(
            <ChannelSubscriptionFilter {...props}/>
        );

        let isValid;
        isValid = wrapper.instance().isValid();
        expect(isValid).toBe(true);

        wrapper.setProps({
            ...props,
            value: {
                inclusion: FilterFieldInclusion.EMPTY,
                key: 'securitylevel',
                values: [],
            },
        });

        isValid = wrapper.instance().isValid();
        expect(isValid).toBe(true);

        wrapper.setProps({
            ...props,
            value: {
                inclusion: FilterFieldInclusion.INCLUDE_ANY,
                key: 'securitylevel',
                values: [],
            },
        });

        isValid = wrapper.instance().isValid();
        expect(isValid).toBe(true);

        wrapper.setProps({
            ...props,
            value: {
                inclusion: FilterFieldInclusion.EXCLUDE_ANY,
                key: 'securitylevel',
                values: [],
            },
        });

        isValid = wrapper.instance().isValid();
        expect(isValid).toBe(false);
        expect(wrapper.find('.error-text').text()).toEqual('Security level inclusion cannot be "Exclude Any". Note that the default value is now "Empty".');
    });
});
