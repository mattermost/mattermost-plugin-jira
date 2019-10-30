// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import Preferences from 'mattermost-redux/constants/preferences';

import projectMetadata from 'testdata/cloud-get-jira-project-metadata.json';
import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project.json';
import testChannel from 'testdata/channel.json';

import {IssueMetadata, ProjectMetadata} from 'types/model';

import EditChannelSettings from './edit_channel_settings';

describe('components/EditChannelSettings', () => {
    const baseActions = {
        createChannelSubscription: jest.fn().mockResolvedValue({}),
        clearIssueMetadata: jest.fn().mockResolvedValue({}),
        deleteChannelSubscription: jest.fn().mockResolvedValue({}),
        editChannelSubscription: jest.fn().mockResolvedValue({}),
        fetchChannelSubscriptions: jest.fn().mockResolvedValue({}),
        fetchJiraIssueMetadataForProjects: jest.fn().mockResolvedValue({}),
    };

    const channelSubscription = {
        id: 'asxtifxe8jyi9y81htww6ixkiy',
        channel_id: '9f8em5tjjirnpretkzywiqtnur',
        filters: {
            events: ['event_updated_reopened'],
            projects: ['KT'],
            issue_types: ['10004'],
            fields: [{
                key: 'customfield_10005',
                inclusion: 'include_any',
                values: ['10000'],
            }, {
                key: 'customfield_10006',
                inclusion: 'include_any',
                values: ['10007'],
            }, {
                key: 'versions',
                inclusion: 'include_any',
                values: ['10000'],
            }, {
                key: 'customfield_10014',
                inclusion: 'include_any',
                values: ['IDT-24'],
            }],
        },
        name: 'SubTestName',
    };

    const baseProps = {
        ...baseActions,
        channel: testChannel,
        theme: Preferences.THEMES.default,
        jiraProjectMetadata: projectMetadata as ProjectMetadata,
        jiraIssueMetadata: issueMetadata as IssueMetadata,
        channelSubscriptions: [channelSubscription],
        close: jest.fn(),
        selectedSubscription: channelSubscription,
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no subscriptions', () => {
        const props = {...baseProps, channelSubscriptions: [], selectedSubscription: null};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no issue metadata', () => {
        const props = {...baseProps, jiraIssueMetadata: null};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot after fetching issue metadata', async () => {
        const props = {...baseProps};
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );

        expect(wrapper.state().fetchingIssueMetadata).toBe(true);
        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no filters', async () => {
        const sub = {
            ...baseProps.channelSubscriptions[0],
            filters: {events: [], projects: [], issue_types: [], fields: []},
        };
        const props = {
            ...baseProps,
            channelSubscriptions: [sub],
            selectedSubscription: sub,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );

        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper).toMatchSnapshot();
    });

    test('should change project filter when chosen', async () => {
        const clearIssueMetadata = jest.fn().mockResolvedValue({});
        let fetchJiraIssueMetadataForProjects = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            fetchJiraIssueMetadataForProjects,
            clearIssueMetadata,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );
        wrapper.instance().handleProjectChange('projects', 'TES');
        expect(wrapper.state().filters.projects).toEqual(['TES']);
        expect(wrapper.state().fetchingIssueMetadata).toBe(true);
        expect(fetchJiraIssueMetadataForProjects).toHaveBeenCalled();
        expect(clearIssueMetadata).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper.state().getMetaDataErr).toBe(null);

        fetchJiraIssueMetadataForProjects = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        wrapper.setProps({fetchJiraIssueMetadataForProjects});

        wrapper.instance().handleProjectChange('projects', 'TES');
        expect(wrapper.state().filters.projects).toEqual(['TES']);
        expect(fetchJiraIssueMetadataForProjects).toHaveBeenCalled();
        expect(wrapper.state().fetchingIssueMetadata).toBe(true);

        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper.state().getMetaDataErr).toEqual('The project TES is unavailable. Please contact your system administrator.');
    });

    test('should create a named subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        let close = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            close,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );

        wrapper.setState({
            filters: channelSubscription.filters,
            subscriptionName: channelSubscription.name,
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(createChannelSubscription).toHaveBeenCalledWith(
            {
                channel_id: testChannel.id,
                filters: channelSubscription.filters,
                name: channelSubscription.name,
            }
        );
        expect(editChannelSubscription).not.toHaveBeenCalled();
        expect(close).not.toHaveBeenCalled();

        await Promise.resolve();
        expect(close).toHaveBeenCalled();

        close = jest.fn();
        wrapper.setProps({
            close,
            createChannelSubscription: jest.fn().mockResolvedValue({error: {message: 'Failure'}}),
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toEqual(null);

        await Promise.resolve();
        expect(close).not.toHaveBeenCalled();
        expect(wrapper.state().error).toEqual('Failure');
    });

    test('SERVER - should create a subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        let close = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            channelSubscriptions: [],
            selectedSubscription: null,
            jiraIssueMetadata: serverIssueMetadata as IssueMetadata,
            jiraProjectMetadata: serverProjectMetadata as ProjectMetadata,
            close,
        };

        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );

        wrapper.setState({
            filters: channelSubscription.filters,
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(createChannelSubscription).toHaveBeenCalledWith(
            {
                channel_id: testChannel.id,
                filters: channelSubscription.filters,
                name: null,
            }
        );
        expect(editChannelSubscription).not.toHaveBeenCalled();
        expect(close).not.toHaveBeenCalled();

        await Promise.resolve();
        expect(close).toHaveBeenCalled();

        close = jest.fn();
        wrapper.setProps({
            close,
            createChannelSubscription: jest.fn().mockResolvedValue({error: {message: 'Failure'}}),
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toEqual(null);

        await Promise.resolve();
        expect(close).not.toHaveBeenCalled();
        expect(wrapper.state().error).toEqual('Failure');
    });

    test('should edit a subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        let close = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            close,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );

        wrapper.setState({
            filters: channelSubscription.filters,
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(editChannelSubscription).toHaveBeenCalledWith(
            {
                id: channelSubscription.id,
                channel_id: testChannel.id,
                filters: channelSubscription.filters,
                name: channelSubscription.name,
            }
        );
        expect(createChannelSubscription).not.toHaveBeenCalled();
        expect(close).not.toHaveBeenCalled();

        await Promise.resolve();
        expect(close).toHaveBeenCalled();

        close = jest.fn();
        wrapper.setProps({
            close,
            editChannelSubscription: jest.fn().mockResolvedValue({error: {message: 'Failure'}}),
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});

        await Promise.resolve();
        expect(close).not.toHaveBeenCalled();
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
        wrapper.instance().handleProjectChange(['KT']);
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

        const filters = channelSubscription.filters;

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

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);
        expect(wrapper.find('#jira-delete-subscription').prop('disabled')).toBe(true);
    });

    test('should delete subscription', async () => {
        const deleteChannelSubscription = jest.fn().mockResolvedValue({});
        const close = jest.fn();
        const props = {
            ...baseProps,
            deleteChannelSubscription,
            close,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);
        wrapper.find('#jira-delete-subscription').simulate('click');

        expect(wrapper.state().showConfirmModal).toBe(true);
        wrapper.instance().handleConfirmDelete();

        expect(deleteChannelSubscription).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().error).toBe(null);
        expect(close).toHaveBeenCalled();
    });

    test('should show error if delete fails', async () => {
        const deleteChannelSubscription = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        const close = jest.fn();
        const props = {
            ...baseProps,
            deleteChannelSubscription,
            close,
        };
        const wrapper = shallow<EditChannelSettings>(
            <EditChannelSettings {...props}/>
        );

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);
        wrapper.find('#jira-delete-subscription').simulate('click');

        expect(wrapper.state().showConfirmModal).toBe(true);
        wrapper.instance().handleConfirmDelete();

        expect(deleteChannelSubscription).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().error).toEqual('Failure');
        expect(close).not.toHaveBeenCalled();
    });
});
