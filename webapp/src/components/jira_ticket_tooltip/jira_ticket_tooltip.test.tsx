// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import {Instance, InstanceType} from 'types/model';

import TicketPopover, {Props} from './jira_ticket_tooltip';

const mockStore = configureStore([thunk]);

const defaultMockState = {
    'plugins-jira': {
        installedInstances: [],
        connectedInstances: [],
    },
    entities: {
        general: {
            config: {
                SiteURL: 'http://localhost:8065',
            },
        },
    },
};

const renderWithRedux = (ui: React.ReactElement, initialState = defaultMockState) => {
    const store = mockStore(initialState);
    return {
        store,
        ...render(
            <IntlProvider locale='en'>
                <Provider store={store}>{ui}</Provider>
            </IntlProvider>,
        ),
    };
};

describe('components/jira_ticket_tooltip', () => {
    describe('getIssueKey', () => {
        const mockConnectedInstances: Instance[] = [
            {
                instance_id: 'https://something-1.atlassian.net',
                type: InstanceType.CLOUD,
            },
            {
                instance_id: 'https://something-2.atlassian.net',
                type: InstanceType.SERVER,
            },
        ];

        const mockProps1: Props = {
            href: '',
            show: false,
            connected: false,
            connectedInstances: mockConnectedInstances,
            fetchIssueByKey: jest.fn(),
        };

        const mockProps2: Props = {
            href: '',
            show: false,
            connected: false,
            connectedInstances: [],
            fetchIssueByKey: jest.fn(),
        };

        beforeEach(() => {
            jest.clearAllMocks();
        });

        test('should return the expected output when URL matches the first regex pattern', async () => {
            const ref = React.createRef<TicketPopover>();
            await act(async () => {
                renderWithRedux(
                    <TicketPopover
                        {...mockProps1}
                        href='https://something-1.atlassian.net/browse/TICKET-1234'
                        ref={ref}
                    />,
                );
            });

            const expectedOutput = {ticketID: 'TICKET-1234', instanceID: 'https://something-1.atlassian.net'};
            expect(ref.current?.getIssueKey()).toEqual(expectedOutput);
        });

        test('should return the expected output when URL matches the second regex pattern', async () => {
            const ref = React.createRef<TicketPopover>();
            await act(async () => {
                renderWithRedux(
                    <TicketPopover
                        {...mockProps1}
                        href='https://something-2.atlassian.net/jira/issues/?selectedIssue=TICKET-1234'
                        ref={ref}
                    />,
                );
            });

            const expectedOutput = {ticketID: 'TICKET-1234', instanceID: 'https://something-2.atlassian.net'};
            expect(ref.current?.getIssueKey()).toEqual(expectedOutput);
        });

        test('should return null when URL does not match any pattern', async () => {
            const ref = React.createRef<TicketPopover>();
            await act(async () => {
                renderWithRedux(
                    <TicketPopover
                        {...mockProps1}
                        href='https://something-invalid.atlassian.net/not-a-ticket'
                        ref={ref}
                    />,
                );
            });

            expect(ref.current?.getIssueKey()).toEqual(null);
        });

        test('should return null when the URL does not contain the ticket ID', async () => {
            const ref = React.createRef<TicketPopover>();
            await act(async () => {
                renderWithRedux(
                    <TicketPopover
                        {...mockProps1}
                        href='https://something-2.atlassian.net/jira/issues/?selectedIssue='
                        ref={ref}
                    />,
                );
            });

            expect(ref.current?.getIssueKey()).toEqual(null);
        });

        test('should return null when no instance is connected', async () => {
            const ref = React.createRef<TicketPopover>();
            await act(async () => {
                renderWithRedux(
                    <TicketPopover
                        {...mockProps2}
                        href='https://something-2.atlassian.net/jira/issues/?selectedIssue='
                        ref={ref}
                    />,
                );
            });

            expect(ref.current?.getIssueKey()).toEqual(null);
        });
    });
});
