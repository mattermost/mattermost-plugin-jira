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
        const props = {...baseProps, issueMetadata: {}};
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
        const {rerender} = await act(async () => {
            return renderWithRedux(
                <ChannelSubscriptionFilter
                    {...props}
                    ref={ref}
                />,
            );
        });

        // Initially no JiraEpicSelector
        expect(ref.current).toBeDefined();

        // After setting Epic Link field
        await act(async () => {
            rerender(
                <IntlProvider locale='en'>
                    <Provider store={mockStore(defaultMockState)}>
                        <ChannelSubscriptionFilter
                            {...props}
                            field={fields.find(isEpicLinkField) as FilterField}
                            ref={ref}
                        />
                    </Provider>
                </IntlProvider>,
            );
        });

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

        const formatOptionLabel = ref.current?.formatInclusionLabel;
        if (formatOptionLabel) {
            const tests = [
                ['include_any', 'Includes either of the values (or)'],
                ['include_all', 'Includes all of the values (and)'],
                ['exclude_any', 'Excludes all of the values'],
                ['empty', 'Includes when the value is empty'],
            ];

            // Select dropdown is open
            for (const t of tests) {
                const element = formatOptionLabel({value: t[0]}, {});
                const wrapper = render(<>{element}</>);
                expect(wrapper.container.textContent).toEqual(t[1]);
                wrapper.unmount();
            }

            // Select dropdown is closed
            const result = formatOptionLabel({value: 'include_any', label: 'Some Option Label'}, {context: 'value'});
            expect(result).toEqual('Some Option Label');
        }
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
        const {rerender} = await act(async () => {
            return renderWithRedux(
                <ChannelSubscriptionFilter
                    {...props}
                    ref={ref}
                />,
            );
        });

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
        const ref = React.createRef<ChannelSubscriptionFilter>();
        const {rerender, container} = await act(async () => {
            return renderWithRedux(
                <ChannelSubscriptionFilter
                    {...props}
                    ref={ref}
                />,
            );
        });

        let isValid;
        isValid = ref.current?.isValid();
        expect(isValid).toBe(true);

        await act(async () => {
            rerender(
                <IntlProvider locale='en'>
                    <Provider store={mockStore(defaultMockState)}>
                        <ChannelSubscriptionFilter
                            {...props}
                            value={{
                                inclusion: FilterFieldInclusion.EMPTY,
                                key: 'securitylevel',
                                values: [],
                            }}
                            ref={ref}
                        />
                    </Provider>
                </IntlProvider>,
            );
        });

        isValid = ref.current?.isValid();
        expect(isValid).toBe(true);

        await act(async () => {
            rerender(
                <IntlProvider locale='en'>
                    <Provider store={mockStore(defaultMockState)}>
                        <ChannelSubscriptionFilter
                            {...props}
                            value={{
                                inclusion: FilterFieldInclusion.INCLUDE_ANY,
                                key: 'securitylevel',
                                values: [],
                            }}
                            ref={ref}
                        />
                    </Provider>
                </IntlProvider>,
            );
        });

        isValid = ref.current?.isValid();
        expect(isValid).toBe(true);

        await act(async () => {
            rerender(
                <IntlProvider locale='en'>
                    <Provider store={mockStore(defaultMockState)}>
                        <ChannelSubscriptionFilter
                            {...props}
                            value={{
                                inclusion: FilterFieldInclusion.EXCLUDE_ANY,
                                key: 'securitylevel',
                                values: [],
                            }}
                            ref={ref}
                        />
                    </Provider>
                </IntlProvider>,
            );
        });

        await act(async () => {
            isValid = ref.current?.isValid();
        });
        expect(isValid).toBe(false);
        expect(container.querySelector('.error-text')?.textContent).toEqual('Security level inclusion cannot be "Exclude Any". Note that the default value is now "Empty".');
    });
});
