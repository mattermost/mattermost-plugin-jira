// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow, mount} from 'enzyme';

import Preferences from 'mattermost-redux/constants/preferences';

import projectMetadata from 'testdata/cloud-get-jira-project-metadata.json';
import issueMetadata from 'testdata/cloud-get-create-issue-metadata-for-project.json';
import serverProjectMetadata from 'testdata/server-get-jira-project-metadata.json';
import serverIssueMetadata from 'testdata/server-get-create-issue-metadata-for-project.json';
import testChannel from 'testdata/channel.json';

import ChannelSettingsInner from './channel_settings_internal';

describe('components/ChannelSettingsInner', () => {
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
            issue_types: ['10001'],
        },
    };

    const baseProps = {
        ...baseActions,
        channel: testChannel,
        theme: Preferences.THEMES.default,
        jiraProjectMetadata: projectMetadata,
        jiraIssueMetadata: issueMetadata,
        channelSubscriptions: [channelSubscription],
        close: jest.fn(),
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no subscriptions', () => {
        const props = {...baseProps, channelSubscriptions: []};
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no issue metadata', () => {
        const props = {...baseProps, jiraIssueMetadata: null};
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot after fetching issue metadata', async () => {
        const props = {...baseProps};
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );

        expect(wrapper.state().fetchingIssueMetadata).toBe(true);
        await Promise.resolve();
        expect(wrapper.state().fetchingIssueMetadata).toBe(false);
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with no filters', async () => {
        const props = {
            ...baseProps,
            channelSubscriptions: [{
                ...baseProps.channelSubscriptions[0],
                filters: {events: [], projects: [], issue_types: []},
            }],
        };
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
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
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
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
        expect(wrapper.state().getMetaDataErr).toEqual('Failure');
    });

    test('should create a subscription', async () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const editChannelSubscription = jest.fn().mockResolvedValue({});
        let close = jest.fn();
        const props = {
            ...baseProps,
            createChannelSubscription,
            editChannelSubscription,
            channelSubscriptions: [],
            close,
        };
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );

        wrapper.setState({
            filters: channelSubscription.filters,
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(createChannelSubscription).toHaveBeenCalledWith({channel_id: testChannel.id, filters: channelSubscription.filters});
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
            jiraIssueMetadata: serverIssueMetadata,
            jiraProjectMetadata: serverProjectMetadata,
            close,
        };
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );

        wrapper.setState({
            filters: channelSubscription.filters,
        });
        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(createChannelSubscription).toHaveBeenCalledWith({channel_id: testChannel.id, filters: channelSubscription.filters});
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
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );

        wrapper.setState({
            filters: channelSubscription.filters,
        });

        wrapper.instance().handleCreate({preventDefault: jest.fn()});
        expect(wrapper.state().error).toBe(null);
        expect(editChannelSubscription).toHaveBeenCalledWith({id: channelSubscription.id, channel_id: testChannel.id, filters: channelSubscription.filters});
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

    test('should not create when choices are blank', () => {
        const createChannelSubscription = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            createChannelSubscription,
            channelSubscriptions: [],
        };
        const wrapper = mount(
            <ChannelSettingsInner {...props}/>
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
        };
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );

        expect(wrapper.exists('#jira-delete-subscription')).toBe(false);
    });

    test('should delete subscription', async () => {
        let deleteChannelSubscription = jest.fn().mockResolvedValue({});
        let close = jest.fn();
        const props = {
            ...baseProps,
            deleteChannelSubscription,
            close,
        };
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);

        wrapper.find('#jira-delete-subscription').simulate('click');
        expect(deleteChannelSubscription).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().error).toBe(null);
        expect(close).toHaveBeenCalled();

        deleteChannelSubscription = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        close = jest.fn();
        wrapper.setProps({deleteChannelSubscription, close});

        wrapper.find('#jira-delete-subscription').simulate('click');
        expect(deleteChannelSubscription).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().error).toEqual('Failure');
        expect(close).not.toHaveBeenCalled();
    });

    test('should show error if delete fails', async () => {
        const deleteChannelSubscription = jest.fn().mockResolvedValue({error: {message: 'Failure'}});
        const close = jest.fn();
        const props = {
            ...baseProps,
            deleteChannelSubscription,
            close,
        };
        const wrapper = shallow(
            <ChannelSettingsInner {...props}/>
        );

        expect(wrapper.exists('#jira-delete-subscription')).toBe(true);
        wrapper.find('#jira-delete-subscription').simulate('click');

        expect(deleteChannelSubscription).toHaveBeenCalled();

        await Promise.resolve();
        expect(wrapper.state().error).toEqual('Failure');
        expect(close).not.toHaveBeenCalled();
    });
});
