// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import JiraInstanceSelector from './jira_instance_selector';

describe('components/JiraInstanceSelector', () => {
    const baseActions = {
        getConnected: jest.fn().mockResolvedValue({}),
    };

    const instance1 = {
        instance_id: 'https://something.atlassian.net',
        is_default: true,
        type: 'cloud' as 'cloud',
    };

    const instance2 = {
        instance_id: 'http://localhost:8080',
        is_default: false,
        type: 'server' as 'server',
    };

    const baseProps = {
        ...baseActions,
        onlyShowConnectedInstances: false,
        instances: [instance1, instance2],
        connectedInstances: [instance1],
        theme: {},
        value: '',
        onChange: jest.fn().mockResolvedValue({}),
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<JiraInstanceSelector>(
            <JiraInstanceSelector {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot when value is set', () => {
        const props = {
            ...baseProps,
            value: instance1.instance_id,
        };
        const wrapper = shallow<JiraInstanceSelector>(
            <JiraInstanceSelector {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should match snapshot when onlyShowConnectedInstances is true', () => {
        const props = {
            ...baseProps,
            onlyShowConnectedInstances: true,
        };
        const wrapper = shallow<JiraInstanceSelector>(
            <JiraInstanceSelector {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });
});
