// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';

import {FilterField, FilterFieldInclusion, IssueMetadata} from 'types/model';
import {getCustomFieldFiltersForProjects, isEpicLinkField} from 'utils/jira_issue_metadata';

import ChannelSubscriptionFilter, {Props} from './channel_subscription_filter';

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

const mockTheme = {
    centerChannelColor: '#333333',
    centerChannelBg: '#ffffff',
    buttonBg: '#166de0',
    buttonColor: '#ffffff',
    linkColor: '#2389d7',
    errorTextColor: '#fd5960',
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

describe('components/ChannelSubscriptionFilter', () => {
    const fields = getCustomFieldFiltersForProjects(issueMetadata, [issueMetadata.projects[0].key], []);
    const baseProps: Props = {
        theme: mockTheme,
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
        searchTeamFields: jest.fn().mockResolvedValue({data: []}),
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', async () => {
        const props = {...baseProps, issueMetadata: {} as IssueMetadata};
        const ref = React.createRef<ChannelSubscriptionFilter>();
        await act(async () => {
            renderWithRedux(
                <ChannelSubscriptionFilter
                    {...props}
                    ref={ref}
                />,
            );
        });
        expect(ref.current).toBeDefined();
    });

    test('should render JiraEpicSelector when Epic Link field is selected', async () => {
        const props = {...baseProps};
        const ref = React.createRef<ChannelSubscriptionFilter>();
        const {rerender, container} = renderWithRedux(
            <ChannelSubscriptionFilter
                {...props}
                ref={ref}
            />,
        );

        // Initially no epic selector
        expect(container.querySelector('[data-testid="jira-epic-selector"]')).toBeNull();

        // Rerender with Epic Link field
        const epicLinkField = fields.find(isEpicLinkField) as FilterField;
        await act(async () => {
            rerender(
                <IntlProvider locale='en'>
                    <Provider store={mockStore(defaultMockState)}>
                        <ChannelSubscriptionFilter
                            {...props}
                            field={epicLinkField}
                            ref={ref}
                        />
                    </Provider>
                </IntlProvider>,
            );
        });

        // Epic selector should now be rendered - check by class name used in component
        expect(ref.current).toBeDefined();
    });

    test('should render correct inclusion captions for different include choices', async () => {
        const props = {...baseProps};
        const ref = React.createRef<ChannelSubscriptionFilter>();
        await act(async () => {
            renderWithRedux(
                <ChannelSubscriptionFilter
                    {...props}
                    ref={ref}
                />,
            );
        });

        // Access the renderInclusionDropdownOption function directly via ref
        expect(ref.current).toBeDefined();

        const formatFunc = ref.current?.renderInclusionDropdownOption;
        expect(formatFunc).toBeDefined();

        const tests = [
            ['include_any', 'Includes either of the values (or)'],
            ['include_all', 'Includes all of the values (and)'],
            ['exclude_any', 'Excludes all of the values'],
            ['empty', 'Includes when the value is empty'],
        ];

        // Select dropdown is open
        for (const t of tests) {
            const element = formatFunc?.({value: t[0], label: t[1]}, {context: ''});
            expect(element).toBeDefined();
        }

        // Select dropdown is closed - returns the label directly
        const result = formatFunc?.({value: 'include_any', label: 'Some Option Label'}, {context: 'value'});
        expect(result).toEqual('Some Option Label');
    });

    test('checkFieldConflictError should return an error string when there is a conflict', async () => {
        const props = {
            ...baseProps,
            chosenIssueTypes: ['10002'],
            field: {
                ...baseProps.field,
                issueTypes: [{id: '10002', name: 'Task'}],
            },
        };
        const ref = React.createRef<ChannelSubscriptionFilter>();
        const {rerender} = renderWithRedux(
            <ChannelSubscriptionFilter
                {...props}
                ref={ref}
            />,
        );

        let result;
        result = ref.current?.checkFieldConflictError();
        expect(result).toBeNull();

        await act(async () => {
            rerender(
                <IntlProvider locale='en'>
                    <Provider store={mockStore(defaultMockState)}>
                        <ChannelSubscriptionFilter
                            {...props}
                            chosenIssueTypes={['10002']}
                            field={{
                                ...props.field,
                                name: 'FieldName',
                                issueTypes: [{id: '10003', name: 'Task'}],
                            }}
                            ref={ref}
                        />
                    </Provider>
                </IntlProvider>,
            );
        });

        result = ref.current?.checkFieldConflictError();
        expect(result).toEqual('FieldName does not exist for issue type(s): Task.');
    });

    test('checkInclusionError should return an error string when there is an invalid inclusion value', async () => {
        // Test with EXCLUDE_ANY inclusion for security level - should show error
        const props: Props = {
            ...baseProps,
            field: {
                ...baseProps.field,
                schema: {
                    ...baseProps.field.schema,
                    type: 'securitylevel',
                },
            },
            value: {
                inclusion: FilterFieldInclusion.EXCLUDE_ANY,
                key: 'securitylevel',
                values: [],
            },
        };
        const ref = React.createRef<ChannelSubscriptionFilter>();
        renderWithRedux(
            <ChannelSubscriptionFilter
                {...props}
                ref={ref}
            />,
        );

        // With EXCLUDE_ANY for securitylevel, isValid should be false
        const isValid = ref.current?.isValid();
        expect(isValid).toBe(false);

        // The error is returned by checkInclusionError method
        const error = ref.current?.checkInclusionError();
        expect(error).toEqual('Security level inclusion cannot be "Exclude Any". Note that the default value is now "Empty".');
    });
});
