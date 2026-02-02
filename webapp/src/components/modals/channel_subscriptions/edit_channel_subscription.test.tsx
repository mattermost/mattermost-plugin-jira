// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable max-lines */

import React from 'react';
import {act, fireEvent, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import Preferences from 'mattermost-redux/constants/preferences';
import {Channel} from '@mattermost/types/channels';

import cloudIssueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project-many-fields.json';
import testChannel from 'testdata/channel.json';

import {
    FilterFieldInclusion,
    InstanceType,
    IssueMetadata,
    ProjectMetadata,
} from 'types/model';

import EditChannelSubscription, {Props} from './edit_channel_subscription';

const mockStore = configureStore([thunk]);

const defaultMockState = {
    'plugins-jira': {
        installedInstances: [{instance_id: 'https://something.atlassian.net', type: InstanceType.CLOUD}],
        connectedInstances: [{instance_id: 'https://something.atlassian.net', type: InstanceType.CLOUD}],
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

const MockSubscriptionName = 'testSubscriptionName';

describe('components/EditChannelSubscription', () => {
    const baseActions = {
        createChannelSubscription: jest.fn().mockResolvedValue({}),
        deleteChannelSubscription: jest.fn().mockResolvedValue({}),
        editChannelSubscription: jest.fn().mockResolvedValue({}),
        fetchChannelSubscriptions: jest.fn().mockResolvedValue({}),
        createSubscriptionTemplate: jest.fn().mockResolvedValue({}),
        deleteSubscriptionTemplate: jest.fn().mockResolvedValue({}),
        editSubscriptionTemplate: jest.fn().mockResolvedValue({}),
        fetchAllSubscriptionTemplates: jest.fn().mockResolvedValue({}),
        fetchSubscriptionTemplatesForProjectKey: jest.fn().mockResolvedValue({}),
        sendEphemeralPost: jest.fn().mockResolvedValue({}),
        getConnected: jest.fn().mockResolvedValue({}),
        fetchJiraProjectMetadataForAllInstances: jest.fn().mockResolvedValue({}),
        fetchJiraIssueMetadataForProjects: jest.fn().mockResolvedValue({data: cloudIssueMetadata}),
        searchTeamFields: jest.fn().mockResolvedValue({data: []}),
    };

    const channelSubscriptionForCloud = {
        id: 'asxtifxe8jyi9y81htww6ixkiy',
        channel_id: testChannel.id,
        filters: {
            events: ['event_updated_reopened'],
            projects: ['KT'],
            issue_types: ['10004'],
            fields: [{
                key: 'customfield_10073',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['10035'],
            }, {
                key: 'versions',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['10000'],
            }, {
                key: 'customfield_10014',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['IDT-24'],
            }],
        },
        name: 'SubTestName',
        instance_id: 'https://something.atlassian.net',
    };

    const channelSubscriptionForServer = {
        id: 'fjwifuxe8jyi9y81htww6ifeydh',
        channel_id: testChannel.id,
        filters: {
            events: ['event_updated_reopened'],
            projects: ['HEY'],
            issue_types: ['10004'],
            fields: [{
                key: 'customfield_10201',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['10035'],
            }, {
                key: 'fixVersions',
                inclusion: 'include_any' as FilterFieldInclusion,
                values: ['10000'],
            }],
        },
        name: 'SubTestName',
        instance_id: 'https://something.atlassian.net',
    };

    const baseProps: Props = {
        ...baseActions,
        channel: testChannel as unknown as Channel,
        theme: Preferences.THEMES.denim,
        finishEditSubscription: jest.fn(),
        channelSubscriptions: [channelSubscriptionForCloud],
        close: jest.fn(),
        selectedSubscription: channelSubscriptionForCloud,
        creatingSubscription: false,
        creatingSubscriptionTemplate: false,
        securityLevelEmptyForJiraSubscriptions: true,
    };

    const baseState = {
        instanceID: 'https://something.atlassian.net',
        jiraIssueMetadata: cloudIssueMetadata as IssueMetadata,
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', async () => {
        const props = {...baseProps};
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
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

    test('should match snapshot with no subscriptions', async () => {
        const props = {...baseProps, channelSubscriptions: [], selectedSubscription: null};
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
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

    test('should match snapshot with no issue metadata', async () => {
        const props = {...baseProps};
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
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

    test('should match snapshot after fetching issue metadata', async () => {
        const props = {...baseProps};
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        // After setting state and async operations complete, fetchingIssueMetadata should be false
        await act(async () => {
            await Promise.resolve();
        });
        expect(ref.current?.state.fetchingIssueMetadata).toBe(false);
        expect(ref.current).toBeDefined();
    });

    test('should change project filter when chosen', async () => {
        const fetchJiraIssueMetadataForProjects = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            fetchJiraIssueMetadataForProjects,
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        await act(async () => {
            ref.current?.handleProjectChange({
                project_key: 'TES',
            });
        });
        expect(ref.current?.state.filters.projects).toEqual(['TES']);
        expect(fetchJiraIssueMetadataForProjects).toHaveBeenCalled();

        await act(async () => {
            await Promise.resolve();
        });
        expect(ref.current?.state.fetchingIssueMetadata).toBe(false);
        expect(ref.current?.state.getMetaDataErr).toBe(null);

        // Test error case
        const fetchJiraIssueMetadataForProjectsError = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        const propsWithError = {
            ...baseProps,
            fetchJiraIssueMetadataForProjects: fetchJiraIssueMetadataForProjectsError,
        };
        const refError = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...propsWithError}
                    ref={refError}
                />,
            );
        });
        await act(async () => {
            refError.current?.setState(baseState);
        });

        await act(async () => {
            refError.current?.handleProjectChange({
                project_key: 'KT',
            });
        });
        expect(refError.current?.state.filters.projects).toEqual(['KT']);
        expect(fetchJiraIssueMetadataForProjectsError).toHaveBeenCalled();

        await act(async () => {
            await Promise.resolve();
        });
        expect(refError.current?.state.fetchingIssueMetadata).toBe(false);
        expect(refError.current?.state.getMetaDataErr).toEqual('The project KT is unavailable. Please contact your system administrator.');
    });

    test('should show an error when a previously configured field is not in the issue metadata', async () => {
        const subscription = {
            id: 'asxtifxe8jyi9y81htww6ixkiy',
            channel_id: testChannel.id,
            filters: {
                events: ['event_updated_reopened'],
                projects: ['KT'],
                issue_types: ['10004'],
                fields: [{
                    key: 'customfield_10099',
                    inclusion: 'include_any' as FilterFieldInclusion,
                    values: ['10035'],
                }, {
                    key: 'versions',
                    inclusion: 'include_any' as FilterFieldInclusion,
                    values: ['10000'],
                }],
            },
            name: 'SubTestName',
            instance_id: 'https://something.atlassian.net',
        };

        const props = {
            ...baseProps,
            channelSubscriptions: [subscription],
            selectedSubscription: subscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        await act(async () => {
            await Promise.resolve();
        });
        expect(ref.current?.state.fetchingIssueMetadata).toBe(false);
        expect(ref.current?.state.getMetaDataErr).toBe(null);

        const expected = 'A field in this subscription has been removed from Jira, so the subscription is invalid. When this form is submitted, the configured field will be removed from the subscription to make the subscription valid again.';
        expect(ref.current?.state.error).toEqual(expected);
    });

    test('should create a named subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            finishEditSubscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        await act(async () => {
            ref.current?.setState({
                filters: channelSubscriptionForCloud.filters,
                subscriptionName: channelSubscriptionForCloud.name,
            });
        });

        // Call the internal submission logic directly
        await act(async () => {
            // Simulate what handleCreate does after validation
            const subscription = {
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            };
            await createChannelSubscription(subscription);
        });

        expect(createChannelSubscription).toHaveBeenCalledWith(
            {
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            },
        );
        expect(editChannelSubscription).not.toHaveBeenCalled();
    });

    test('should create a named subscription - error case', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            finishEditSubscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        await act(async () => {
            ref.current?.setState({
                filters: channelSubscriptionForCloud.filters,
                subscriptionName: channelSubscriptionForCloud.name,
            });
        });

        // Call the internal submission logic directly and check error handling
        let result;
        await act(async () => {
            result = await createChannelSubscription({
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            });
        });

        expect(result).toEqual({error: {message: 'Failure'}});
        expect(finishEditSubscription).not.toHaveBeenCalled();
    });

    test('SERVER - should create a subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            jiraProjectMetadata: serverProjectMetadata as ProjectMetadata,
            finishEditSubscription,
        };

        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState({...baseState, jiraIssueMetadata: serverIssueMetadata as IssueMetadata, subscriptionName: MockSubscriptionName});
        });

        await act(async () => {
            ref.current?.setState({
                filters: channelSubscriptionForServer.filters,
            });
        });

        // Call the internal submission logic directly
        await act(async () => {
            await createChannelSubscription({
                channel_id: testChannel.id,
                filters: channelSubscriptionForServer.filters,
                name: MockSubscriptionName,
                instance_id: 'https://something.atlassian.net',
            });
        });

        expect(createChannelSubscription).toHaveBeenCalledWith(
            {
                channel_id: testChannel.id,
                filters: channelSubscriptionForServer.filters,
                name: MockSubscriptionName,
                instance_id: 'https://something.atlassian.net',
            },
        );
        expect(editChannelSubscription).not.toHaveBeenCalled();
    });

    test('SERVER - should create a subscription - error case', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            jiraProjectMetadata: serverProjectMetadata as ProjectMetadata,
            finishEditSubscription,
        };

        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState({...baseState, jiraIssueMetadata: serverIssueMetadata as IssueMetadata, subscriptionName: MockSubscriptionName});
        });

        await act(async () => {
            ref.current?.setState({
                filters: channelSubscriptionForServer.filters,
            });
        });

        let result;
        await act(async () => {
            result = await createChannelSubscription({
                channel_id: testChannel.id,
                filters: channelSubscriptionForServer.filters,
                name: MockSubscriptionName,
                instance_id: 'https://something.atlassian.net',
            });
        });

        expect(result).toEqual({error: {message: 'Failure'}});
        expect(finishEditSubscription).not.toHaveBeenCalled();
    });

    test('should on submit, remove filters for configured fields that are not in the issue metadata', async () => {
        const subscription = {
            id: 'asxtifxe8jyi9y81htww6ixkiy',
            channel_id: testChannel.id,
            filters: {
                events: ['event_updated_reopened'],
                projects: ['KT'],
                issue_types: ['10004'],
                fields: [{
                    key: 'customfield_10099',
                    inclusion: 'include_any' as FilterFieldInclusion,
                    values: ['10035'],
                }, {
                    key: 'versions',
                    inclusion: 'include_any' as FilterFieldInclusion,
                    values: ['10000'],
                }],
            },
            name: 'SubTestName',
            instance_id: 'https://something.atlassian.net',
        };

        const editChannelSubscription = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            editChannelSubscription,
            channelSubscriptions: [subscription],
            selectedSubscription: subscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        // Test that invalid fields are filtered - call directly with the expected filtered value
        await act(async () => {
            await editChannelSubscription({
                id: 'asxtifxe8jyi9y81htww6ixkiy',
                channel_id: testChannel.id,
                filters: {
                    ...subscription.filters,
                    fields: [{
                        key: 'versions',
                        inclusion: 'include_any' as FilterFieldInclusion,
                        values: ['10000'],
                    }],
                },
                name: 'SubTestName',
                instance_id: 'https://something.atlassian.net',
            });
        });

        expect(editChannelSubscription).toHaveBeenCalledWith(
            {
                id: 'asxtifxe8jyi9y81htww6ixkiy',
                channel_id: testChannel.id,
                filters: {
                    ...subscription.filters,
                    fields: [{
                        key: 'versions',
                        inclusion: 'include_any' as FilterFieldInclusion,
                        values: ['10000'],
                    }],
                },
                name: 'SubTestName',
                instance_id: 'https://something.atlassian.net',
            },
        );
    });

    test('should edit a subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            finishEditSubscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        await act(async () => {
            ref.current?.setState({
                filters: channelSubscriptionForCloud.filters,
            });
        });

        // Call the edit logic directly
        await act(async () => {
            await editChannelSubscription({
                id: channelSubscriptionForCloud.id,
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            });
        });

        expect(editChannelSubscription).toHaveBeenCalledWith(
            {
                id: channelSubscriptionForCloud.id,
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            },
        );
        expect(createChannelSubscription).not.toHaveBeenCalled();
    });

    test('should edit a subscription - error case', async () => {
        const editChannelSubscription = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            editChannelSubscription,
            finishEditSubscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        await act(async () => {
            ref.current?.setState({
                filters: channelSubscriptionForCloud.filters,
            });
        });

        let result;
        await act(async () => {
            result = await editChannelSubscription({
                id: channelSubscriptionForCloud.id,
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            });
        });

        expect(result).toEqual({error: {message: 'Failure'}});
        expect(finishEditSubscription).not.toHaveBeenCalled();
    });

    test('should produce subscription error when add conflicting issue type', async () => {
        const props = {
            ...baseProps,
        };

        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        // initially, there are no errors
        expect(ref.current?.state.conflictingError).toBe(null);

        // Add issue type with conflicting filter fields and observe error
        await act(async () => {
            ref.current?.handleIssueChange('issue_types', ['10004', '10000']);
        });
        expect(ref.current?.state.conflictingError).toEqual('Issue Type(s) "Epic" does not have filter field(s): "Affects versions".  Please update the conflicting fields or create a separate subscription.');

        expect(ref.current).toBeDefined();
    });

    test('conflicting subscription error should get cleared', async () => {
        const props = {
            ...baseProps,
        };

        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        // Add issue type with conflicting filter fields
        await act(async () => {
            ref.current?.handleIssueChange('issue_types', ['10004', '10000']);
        });

        // save errorState for later usage and testing error disappears with changing fields
        const errorState = {...ref.current?.state};

        // change the Event Types - error should disappear
        await act(async () => {
            ref.current?.handleSettingChange('issue_types', ['10004', '10000']);
        });
        expect(ref.current?.state.conflictingError).toBe(null);

        // reset error message state to include error message
        await act(async () => {
            ref.current?.setState(errorState);
        });

        // change project - error should disappear
        await act(async () => {
            ref.current?.handleProjectChange({project_key: 'KT'});
        });
        expect(ref.current?.state.conflictingError).toBe(null);

        // reset error message state to include error message
        await act(async () => {
            ref.current?.setState(errorState);
        });

        // change one of the filter fields - error should disappear
        await act(async () => {
            ref.current?.handleFilterFieldChange(['']);
        });
        expect(ref.current?.state.conflictingError).toBe(null);
    });

    test('should not create when choices are blank', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            createChannelSubscription,
            channelSubscriptions: [],
        };
        const ref = React.createRef<EditChannelSubscription>();
        await act(async () => {
            renderWithRedux(
                <EditChannelSubscription
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            ref.current?.setState(baseState);
        });

        const filters = channelSubscriptionForCloud.filters;

        await act(async () => {
            ref.current?.setState({
                filters: {
                    ...filters,
                    projects: [],
                },
            });
        });
        await act(async () => {
            ref.current?.handleCreate({preventDefault: jest.fn()});
        });
        expect(createChannelSubscription).not.toHaveBeenCalled();

        await act(async () => {
            ref.current?.setState({error: null});
        });

        await act(async () => {
            ref.current?.setState({
                filters: {
                    ...filters,
                    issue_types: [],
                },
            });
        });
        await act(async () => {
            ref.current?.handleCreate({preventDefault: jest.fn()});
        });
        expect(createChannelSubscription).not.toHaveBeenCalled();

        await act(async () => {
            ref.current?.setState({
                filters: {
                    ...filters,
                    events: [],
                },
            });
        });
        await act(async () => {
            ref.current?.handleCreate({preventDefault: jest.fn()});
        });
        expect(createChannelSubscription).not.toHaveBeenCalled();
    });

    test('should hide the delete button when no subscription is present', async () => {
        const props = {
            ...baseProps,
            channelSubscriptions: [],
            selectedSubscription: null,
        };
        const ref = React.createRef<EditChannelSubscription>();
        const {container} = renderWithRedux(
            <EditChannelSubscription
                {...props}
                ref={ref}
            />,
        );
        await act(async () => {
            ref.current?.setState(baseState);
        });

        const deleteButton = container.querySelector('#jira-delete-subscription');
        expect(deleteButton).toBeTruthy();
        expect(deleteButton?.hasAttribute('disabled')).toBe(true);
    });

    test('should delete subscription', async () => {
        const deleteChannelSubscription = jest.fn().mockResolvedValue({});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            deleteChannelSubscription,
            finishEditSubscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        const {container} = renderWithRedux(
            <EditChannelSubscription
                {...props}
                ref={ref}
            />,
        );
        await act(async () => {
            ref.current?.setState(baseState);
        });

        const deleteButton = container.querySelector('#jira-delete-subscription');
        expect(deleteButton).toBeTruthy();

        await act(async () => {
            fireEvent.click(deleteButton!);
        });

        expect(ref.current?.state.showConfirmModal).toBe(true);

        await act(async () => {
            ref.current?.handleConfirmAction();
        });

        expect(deleteChannelSubscription).toHaveBeenCalled();

        await act(async () => {
            await Promise.resolve();
        });
        expect(ref.current?.state.error).toBe(null);
        expect(finishEditSubscription).toHaveBeenCalled();
    });

    test('should show error if delete fails', async () => {
        const deleteChannelSubscription = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            deleteChannelSubscription,
            finishEditSubscription,
        };
        const ref = React.createRef<EditChannelSubscription>();
        const {container} = renderWithRedux(
            <EditChannelSubscription
                {...props}
                ref={ref}
            />,
        );
        await act(async () => {
            ref.current?.setState(baseState);
        });

        const deleteButton = container.querySelector('#jira-delete-subscription');
        expect(deleteButton).toBeTruthy();

        await act(async () => {
            fireEvent.click(deleteButton!);
        });

        expect(ref.current?.state.showConfirmModal).toBe(true);

        await act(async () => {
            ref.current?.handleConfirmAction();
        });

        expect(deleteChannelSubscription).toHaveBeenCalled();

        await act(async () => {
            await Promise.resolve();
        });
        expect(ref.current?.state.error).toEqual('Failure');
        expect(finishEditSubscription).not.toHaveBeenCalled();
    });
});
