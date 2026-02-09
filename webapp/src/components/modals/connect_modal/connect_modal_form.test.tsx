// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act} from '@testing-library/react';

import {InstanceType} from 'types/model';
import {mockTheme, renderWithRedux} from 'testlib/test-utils';

import ConnectModalForm from './connect_modal_form';

describe('components/ConnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        redirectConnect: jest.fn().mockResolvedValue({}),
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

    test('should render component', async () => {
        const props = {...baseProps};
        const ref = React.createRef<ConnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <ConnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        expect(ref.current).toBeDefined();
    });

    test('should redirect on submit', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const redirectConnect = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            closeModal,
            redirectConnect,
        };
        const ref = React.createRef<ConnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <ConnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            ref.current?.handleInstanceChoice('', 'http://localhost:8080');
        });
        expect(ref.current?.state.selectedInstance).toEqual('http://localhost:8080');
        expect(ref.current?.state.error).toEqual('');
    });

    test('should show error when user is already connected to instance', async () => {
        const props = {...baseProps};
        const ref = React.createRef<ConnectModalForm>();
        await act(async () => {
            renderWithRedux(
                <ConnectModalForm
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            ref.current?.handleInstanceChoice('', 'https://something.atlassian.net');
        });
        expect(ref.current?.state.selectedInstance).toEqual('https://something.atlassian.net');
        expect(ref.current?.state.error).toEqual('You are already connected to this Jira instance.');
    });
});
