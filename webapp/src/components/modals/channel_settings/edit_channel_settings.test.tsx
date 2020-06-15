// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import Preferences from 'mattermost-redux/constants/preferences';

import cloudProjectMetadata from 'testdata/cloud-get-jira-project-metadata.json';
import cloudIssueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project-many-fields.json';
import testChannel from 'testdata/channel.json';

import {IssueMetadata, ProjectMetadata, FilterFieldInclusion} from 'types/model';

import EditChannelSettings from './edit_channel_settings';

describe('components/EditChannelSettings', () => {
    const baseActions = {
        createChannelSubscription: jest.fn().mockResolvedValue({}),
        deleteChannelSubscription: jest.fn().mockResolvedValue({}),
        editChannelSubscription: jest.fn().mockResolvedValue({}),
        fetchChannelSubscriptions: jest.fn().mockResolvedValue({}),
        fetchJiraIssueMetadataForProjects: jest.fn().mockResolvedValue({data: cloudIssueMetadata}),
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

    const baseProps = {
        ...baseActions,
        channel: testChannel,
        theme: Preferences.THEMES.default,
        finishEditSubscription: jest.fn(),
        channelSubscriptions: [channelSubscriptionForCloud],
        close: jest.fn(),
        selectedSubscription: channelSubscriptionForCloud,
        creatingSubscription: false,
    };

    const baseState = {
        instanceID: 'https://something.atlassian.net',
        jiraIssueMetadata: cloudIssueMetadata as IssueMetadata,
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no subscriptions', () => {
        const props = {...baseProps, channelSubscriptions: [], selectedSubscription: null};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no issue metadata', () => {
        const props = {...baseProps};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState({...baseState, jiraIssueMetadata: null});
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot after fetching issue metadata', async () => {
        const props = {...baseProps};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        expect(wrapper.state().fetchingIssueMetadata).toBe(true);
        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper).toMatchSnapshot();
    });

    test('should change project filter when chosen', async () => {
        let fetchJiraIssueMetadataForProjects = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            fetchJiraIssueMetadataForProjects,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);
        wrapper.instance().handleProjectChange('TES');
        expect(wrapper.state().filters.projects).toEqual(['TES']);
        expect(wrapper.state().fetchingIssueMetadata).toBe(true);
        expect(fetchJiraIssueMetadataForProjects).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper.state().getMetaDataErr).toBe(null);

        fetchJiraIssueMetadataForProjects = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        wrapper.setProps({fetchJiraIssueMetadataForProjects});

        wrapper.instance().handleProjectChange('KT');
        expect(wrapper.state().filters.projects).toEqual(['KT']);
        expect(fetchJiraIssueMetadataForProjects).toHaveBeenCalled();
        expect(wrapper.state().fetchingIssueMetadata).toBe(true);

        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper.state().getMetaDataErr).toEqual('The project KT is unavailable. Please contact your system administrator.');
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
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper.state().getMetaDataErr).toBe(null);

        const expected = 'A field in this subscription has been removed from Jira, so the subscription is invalid. When this form is submitted, the configured field will be removed from the subscription to make the subscription valid again.';
        expect(wrapper.state().error).toEqual(expected);
    });

    test('should create a named subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        let finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            finishEditSubscription,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        wrapper.setState({
            filters: channelSubscriptionForCloud.filters,
            subscriptionName: channelSubscriptionForCloud.name,
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(createChannelSubscription).toHaveBeenCalledWith(
            {
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            }
        );
        expect(editChannelSubscription).not.toHaveBeenCalled();
        expect(finishEditSubscription).not.toHaveBeenCalled();

        await Promise.resolve();
        expect(finishEditSubscription).toHaveBeenCalled();

        finishEditSubscription = jest.fn();
        wrapper.setProps({
            finishEditSubscription,
            createChannelSubscription: jest.fn().mockResolvedValue({error: {message: 'Failure'}}),
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toEqual(null);

        await Promise.resolve();
        expect(finishEditSubscription).not.toHaveBeenCalled();
        expect(wrapper.state().error).toEqual('Failure');
    });

    test('SERVER - should create a subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        let finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            jiraProjectMetadata: serverProjectMetadata as ProjectMetadata,
            finishEditSubscription,
        };

        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState({...baseState, jiraIssueMetadata: serverIssueMetadata});

        wrapper.setState({
            filters: channelSubscriptionForServer.filters,
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(createChannelSubscription).toHaveBeenCalledWith(
            {
                channel_id: testChannel.id,
                filters: channelSubscriptionForServer.filters,
                name: null,
                instance_id: 'https://something.atlassian.net',
            }
        );
        expect(editChannelSubscription).not.toHaveBeenCalled();
        expect(finishEditSubscription).not.toHaveBeenCalled();

        await Promise.resolve();
        expect(finishEditSubscription).toHaveBeenCalled();

        finishEditSubscription = jest.fn();
        wrapper.setProps({
            finishEditSubscription,
            createChannelSubscription: jest.fn().mockResolvedValue({error: {message: 'Failure'}}),
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toEqual(null);

        await Promise.resolve();
        expect(finishEditSubscription).not.toHaveBeenCalled();
        expect(wrapper.state().error).toEqual('Failure');
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
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
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
            }
        );
    });

    test('should edit a subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        let finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            finishEditSubscription,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        wrapper.setState({
            filters: channelSubscriptionForCloud.filters,
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(editChannelSubscription).toHaveBeenCalledWith(
            {
                id: channelSubscriptionForCloud.id,
                channel_id: testChannel.id,
                filters: channelSubscriptionForCloud.filters,
                name: channelSubscriptionForCloud.name,
                instance_id: 'https://something.atlassian.net',
            }
        );
        expect(createChannelSubscription).not.toHaveBeenCalled();
        expect(finishEditSubscription).not.toHaveBeenCalled();

        await Promise.resolve();
        expect(finishEditSubscription).toHaveBeenCalled();

        finishEditSubscription = jest.fn();
        wrapper.setProps({
            finishEditSubscription,
            editChannelSubscription: jest.fn().mockResolvedValue({error: {message: 'Failure'}}),
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});

        await Promise.resolve();
        expect(finishEditSubscription).not.toHaveBeenCalled();
        expect(wrapper.state().error).toEqual('Failure');
    });

    test('should produce subscription error when add conflicting issue type', async () => {
        // This test checks that adding an issue type with confilcting fields
        // will trigger an error message that lists the conflicting filter
        // fields.

        const props = {
            ...baseProps,
        };

        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        // initially, there are no errors
        expect(wrapper.state().conflictingError).toBe(null);

        // Add issue type with conflicting filter fields and observe error
        wrapper.instance().handleIssueChange('issue_types', ['10004', '10000']);
        expect(wrapper.state().conflictingError).toEqual('Issue Type(s) "Epic" does not have filter field(s): "Affects versions".  Please update the conflicting fields or create a separate subscription.');

        // save snapshot showing error message
        expect(wrapper).toMatchSnapshot();
    });

    test('conflicting subscription error should get cleared', async () => {
        // Check that the conflicting error message disappears with
        // each change of a filter field, project, or event

        const props = {
            ...baseProps,
        };

        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        // Add issue type with conflicting filter fields
        wrapper.instance().handleIssueChange('issue_types', ['10004', '10000']);

        // save errorState for later usage and testing error disappears with changing fields
        const errorState = wrapper.state();

        // change the Event Types - error should disappear
        wrapper.instance().handleSettingChange('issue_types', ['10004', '10000']);
        expect(wrapper.state().conflictingError).toBe(null);

        // reset error message state to include error message
        wrapper.setState({...errorState});

        // change project - error should disappear
        wrapper.instance().handleProjectChange('project', 'KT');
        expect(wrapper.state().conflictingError).toBe(null);

        // reset error message state to include error message
        wrapper.setState({...errorState});

        // change one of the filter fields - error should disappear
        wrapper.instance().handleFilterFieldChange(['']);
        expect(wrapper.state().conflictingError).toBe(null);
    });

    test('should not create when choices are blank', () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            createChannelSubscription,
            channelSubscriptions: [],
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        const filters = channelSubscriptionForCloud.filters;

        wrapper.setState({
            filters: {
                ...filters,
                projects: [],
            },
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(createChannelSubscription).not.toHaveBeenCalled();

        wrapper.setState({error: null});

        wrapper.setState({
            filters: {
                ...filters,
                issue_types: [],
            },
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(createChannelSubscription).not.toHaveBeenCalled();

        wrapper.setState({
            filters: {
                ...filters,
                events: [],
            },
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(createChannelSubscription).not.toHaveBeenCalled();
    });

    test('should hide the delete button when no subscription is present', () => {
        const props = {
            ...baseProps,
            channelSubscriptions: [],
            selectedSubscription: null,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);
        expect(wrapper.find('#jira-delete-subscription').prop('disabled')).toBe(true);
    });

    test('should delete subscription', async () => {
        const deleteChannelSubscription = jest.fn().mockResolvedValue({});
        const finishEditSubscription = jest.fn();
        const props = {
            ...baseProps,
            deleteChannelSubscription,
            finishEditSubscription,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);
        wrapper.find('#jira-delete-subscription').simulate('click');

        expect(wrapper.state().showConfirmModal).toBe(true);
        wrapper.instance().handleConfirmDelete();

        expect(deleteChannelSubscription).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().error).toBe(null);
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
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.setState(baseState);

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);
        wrapper.find('#jira-delete-subscription').simulate('click');

        expect(wrapper.state().showConfirmModal).toBe(true);
        wrapper.instance().handleConfirmDelete();

        expect(deleteChannelSubscription).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().error).toEqual('Failure');
        expect(finishEditSubscription).not.toHaveBeenCalled();
    });
});
