// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import ConnectModalForm from './connect_modal_form';

describe('components/ConnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        redirectConnect: jest.fn().mockResolvedValue({}),
    };

    const baseProps = {
        ...baseActions,
        visible: true,
        theme: {},
        connectedInstances: [
            {
                instance_id: 'https://something.atlassian.net',
                is_default: true,
                type: 'cloud' as 'cloud',
            },
        ],
        installedInstances: [
            {
                instance_id: 'https://something.atlassian.net',
                is_default: true,
                type: 'cloud' as 'cloud',
            },
            {
                instance_id: 'http://localhost:8080',
                is_default: true,
                type: 'server' as 'server',
            },
        ],
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<ConnectModalForm>(
            <ConnectModalForm {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should redirect on submit', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const redirectConnect = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            closeModal,
            redirectConnect,
        };
        const wrapper = shallow<ConnectModalForm>(
            <ConnectModalForm {...props}/>
        );

        wrapper.instance().handleInstanceChoice('', 'http://localhost:8080');
        expect(wrapper.state().selectedInstance).toEqual('http://localhost:8080');
        expect(wrapper.state().error).toEqual('');
    });

    test('should show error when user is already connected to instance', async () => {
        const props = {...baseProps};
        const wrapper = shallow<ConnectModalForm>(
            <ConnectModalForm {...props}/>
        );

        wrapper.instance().handleInstanceChoice('', 'https://something.atlassian.net');
        expect(wrapper.state().selectedInstance).toEqual('https://something.atlassian.net');
        expect(wrapper.state().error).toEqual('You are already connected to this Jira instance.');
    });
});
