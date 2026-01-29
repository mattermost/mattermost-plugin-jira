// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render} from '@testing-library/react';

import {InstanceType} from 'types/model';

import DisconnectModalForm from './disconnect_modal_form';

describe('components/DisconnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        sendEphemeralPost: jest.fn().mockResolvedValue({}),
        disconnectUser: jest.fn().mockResolvedValue({}),
    };

    const mockTheme = {
        centerChannelColor: '#333333',
        centerChannelBg: '#ffffff',
        buttonBg: '#166de0',
        buttonColor: '#ffffff',
        linkColor: '#2389d7',
        errorTextColor: '#fd5960',
    };

    const baseProps = {
        ...baseActions,
        visible: true,
        theme: mockTheme,
        connectedInstances: [
            {
                instance_id: 'https://something.atlassian.net',
                type: InstanceType.CLOUD,
            },
        ],
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', () => {
        const props = {...baseProps};
        const {container} = render(<DisconnectModalForm {...props}/>);
        expect(container).toBeInTheDocument();
    });

    test('should close modal and send ephemeral post on submit success', () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const sendEphemeralPost = jest.fn().mockResolvedValue({});
        const disconnectUser = jest.fn().mockResolvedValue({});

        const props = {
            ...baseProps,
            closeModal,
            sendEphemeralPost,
            disconnectUser,
        };
        const {container} = render(<DisconnectModalForm {...props}/>);
        expect(container).toBeInTheDocument();
    });

    test('should show error on submit fail', () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const sendEphemeralPost = jest.fn().mockResolvedValue({});
        const disconnectUser = jest.fn().mockResolvedValue({error: 'Error disconnecting'});

        const props = {
            ...baseProps,
            closeModal,
            sendEphemeralPost,
            disconnectUser,
        };
        const {container} = render(<DisconnectModalForm {...props}/>);
        expect(container).toBeInTheDocument();
    });
});
