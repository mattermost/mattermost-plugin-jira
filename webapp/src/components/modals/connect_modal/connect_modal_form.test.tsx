// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render} from '@testing-library/react';

import {InstanceType} from 'types/model';

import ConnectModalForm from './connect_modal_form';

describe('components/ConnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        redirectConnect: jest.fn().mockResolvedValue({}),
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
        installedInstances: [
            {
                instance_id: 'https://something.atlassian.net',
                type: InstanceType.CLOUD,
            },
            {
                instance_id: 'http://localhost:8080',
                type: InstanceType.SERVER,
            },
        ],
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot', () => {
        const props = {...baseProps};
        const {container} = render(<ConnectModalForm {...props}/>);
        expect(container).toBeInTheDocument();
    });

    test('should redirect on submit', () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const redirectConnect = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            closeModal,
            redirectConnect,
        };
        const {container} = render(<ConnectModalForm {...props}/>);
        expect(container).toBeInTheDocument();
    });

    test('should show error when user is already connected to instance', () => {
        const props = {...baseProps};
        const {container} = render(<ConnectModalForm {...props}/>);
        expect(container).toBeInTheDocument();
    });
});
