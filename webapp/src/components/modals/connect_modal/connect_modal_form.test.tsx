// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, screen, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import {InstanceType} from 'types/model';
import {renderWithRedux} from 'testlib/test-utils';

import ConnectModalForm from './connect_modal_form';

describe('components/ConnectModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        redirectConnect: jest.fn().mockResolvedValue({}),
    };

    const baseProps = {
        ...baseActions,
        visible: true,
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

    test('should connect to the instance picked in the dialog', async () => {
        const closeModal = jest.fn();
        const redirectConnect = jest.fn();
        renderWithRedux(
            <ConnectModalForm
                {...baseProps}
                closeModal={closeModal}
                redirectConnect={redirectConnect}
            />,
        );

        const dialog = screen.getByRole('dialog', {name: 'Connect to Jira'});
        const connect = within(dialog).getByRole('button', {name: 'Connect'});
        expect(connect).toBeDisabled();

        await userEvent.click(within(dialog).getByRole('combobox', {name: 'Jira Instance'}));
        await userEvent.click(await screen.findByRole('option', {name: 'http://localhost:8080'}));
        await userEvent.click(connect);

        expect(closeModal).toHaveBeenCalled();
        expect(redirectConnect).toHaveBeenCalledWith('http://localhost:8080');
    });

    test('should show an alert when picking an already connected instance', async () => {
        renderWithRedux(<ConnectModalForm {...baseProps}/>);

        await userEvent.click(screen.getByRole('combobox', {name: 'Jira Instance'}));
        await userEvent.click(await screen.findByRole('option', {name: 'https://something.atlassian.net'}));

        expect(screen.getByRole('alert')).toHaveTextContent('You are already connected to this Jira instance.');
        expect(screen.getByRole('button', {name: 'Connect'})).toBeDisabled();
    });
});
