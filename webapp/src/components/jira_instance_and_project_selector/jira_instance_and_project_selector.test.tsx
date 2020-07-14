// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import {Theme} from 'mattermost-redux/types/preferences';

import {InstanceType} from 'types/model';

import JiraInstanceAndProjectSelector, {Props} from './jira_instance_and_project_selector';

describe('components/JiraInstanceAndProjectSelector', () => {
    const baseProps: Props = {
        selectedInstanceID: null,
        selectedProjectID: null,
        onInstanceChange: jest.fn(),
        onProjectChange: jest.fn(),
        onError: jest.fn(),

        theme: {} as Theme,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),

        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}, {instance_id: 'instance3', type: InstanceType.SERVER}],
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
        defaultUserInstanceID: '',
        fetchJiraProjectMetadata: jest.fn().mockResolvedValue({data: {
            default_project_key: 'TEST',
            projects: [
                {value: 'TEST', label: 'Test Project'},
                {value: 'AA', label: 'Apples Arrangement'},
            ],
        }}),
        getConnected: jest.fn().mockResolvedValue({error: null}),
        hideProjectSelector: false,
    };

    test('should match snapshot with one connected instance', () => {
        const props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        };
        const wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );

        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with two connected instances', () => {
        const props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
        };
        const wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );

        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot with a default instance selected', () => {
        const props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
            defaultUserInstanceID: 'instance1',
        };
        const wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );

        expect(wrapper).toMatchSnapshot();
    });

    test('should assign the correct initial instance id', async () => {
        let props = {
            ...baseProps,
            onInstanceChange: jest.fn(),
            defaultUserInstanceID: 'instance2',
        };
        let wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );

        await props.getConnected();
        expect(props.onInstanceChange).toBeCalledWith('instance2');

        props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
            onInstanceChange: jest.fn(),
        };
        wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );
        await props.getConnected();
        expect(props.onInstanceChange).toBeCalledWith('instance1');

        props = {
            ...baseProps,
            onInstanceChange: jest.fn(),
            defaultUserInstanceID: 'instance2',
            selectedInstanceID: 'instance3', // pre-selected instance should take precedence. i.e. from existing subscription
        };
        wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );
        await props.getConnected();
        expect(props.onInstanceChange).toBeCalledWith('instance3');

        props = {
            ...baseProps,
            onInstanceChange: jest.fn(),
        };
        wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );
        await props.getConnected();
        expect(props.onInstanceChange).not.toBeCalled();
    });

    test('should use default project key after fetch', async () => {
        const props = {
            ...baseProps,
            defaultUserInstanceID: 'instance2',
            onProjectChange: jest.fn(),
        };
        const wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );
        await props.getConnected();
        expect(wrapper.state().fetchingProjectMetadata).toBe(true);

        await props.fetchJiraProjectMetadata('');
        expect(props.onProjectChange).toBeCalledWith('TEST');
    });

    test('should pass error on failed fetch', async () => {
        const props = {
            ...baseProps,
            fetchJiraProjectMetadata: jest.fn().mockResolvedValue({error: {message: 'Some error'}}),
            onError: jest.fn(),
            defaultUserInstanceID: 'instance2',
        };
        const wrapper = shallow<JiraInstanceAndProjectSelector>(
            <JiraInstanceAndProjectSelector {...props}/>
        );

        await props.getConnected();
        await props.fetchJiraProjectMetadata('');
        expect(props.onError).toHaveBeenCalledWith('Some error');
    });
});
