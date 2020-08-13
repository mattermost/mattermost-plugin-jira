// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import DisconnectModalForm from './disconnect_modal_form';

describe('components/DisconnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        sendEphemeralPost: jest.fn().mockResolvedValue({}),
        disconnectUser: jest.fn().mockResolvedValue({}),
    };

    const baseProps = {
        ...baseActions,
        visible: true,
        theme: {},
        connectedInstances: [],
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<DisconnectModalForm>(
            <DisconnectModalForm {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should close modal and send ephemeral post on submit success', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const sendEphemeralPost = jest.fn().mockResolvedValue({});
        const disconnectUser = jest.fn().mockResolvedValue({});

        const props = {
            ...baseProps,
            closeModal,
            sendEphemeralPost,
            disconnectUser,
        };
        const wrapper = shallow<DisconnectModalForm>(
            <DisconnectModalForm {...props}/>
        );

        wrapper.instance().handleInstanceChoice('', 'https://something.atlassian.net');
        expect(wrapper.state().selectedInstance).toEqual('https://something.atlassian.net');

        wrapper.instance().submit({preventDefault: jest.fn()});
        await Promise.resolve();

        expect(disconnectUser).toHaveBeenCalledWith('https://something.atlassian.net');
        expect(sendEphemeralPost).toHaveBeenCalledWith('Successfully disconnected from Jira instance https://something.atlassian.net');
        expect(closeModal).toHaveBeenCalled();
    });

    test('should show error on submit fail', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const sendEphemeralPost = jest.fn().mockResolvedValue({});
        const disconnectUser = jest.fn().mockResolvedValue({error: 'Error disconnecting'});

        const props = {
            ...baseProps,
            closeModal,
            sendEphemeralPost,
            disconnectUser,
        };
        const wrapper = shallow<DisconnectModalForm>(
            <DisconnectModalForm {...props}/>
        );

        wrapper.instance().handleInstanceChoice('', 'https://something.atlassian.net');
        expect(wrapper.state().selectedInstance).toEqual('https://something.atlassian.net');

        wrapper.instance().submit({preventDefault: jest.fn()});
        await Promise.resolve();

        expect(disconnectUser).toHaveBeenCalled();
        expect(sendEphemeralPost).not.toHaveBeenCalled();
        expect(closeModal).not.toHaveBeenCalled();

        expect(wrapper.state().error).toEqual('Error disconnecting');
    });
});
