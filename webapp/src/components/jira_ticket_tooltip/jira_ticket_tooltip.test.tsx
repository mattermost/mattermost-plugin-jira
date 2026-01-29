// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render} from '@testing-library/react';

import {Instance, InstanceType} from 'types/model';

import TicketPopover, {Props} from './jira_ticket_tooltip';

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

        test('should return the expected output when URL matches the first regex pattern', () => {
            const ref = React.createRef<TicketPopover>();
            render(
                <TicketPopover
                    {...mockProps1}
                    href='https://something-1.atlassian.net/browse/TICKET-1234'
                    ref={ref}
                />,
            );
            const expectedOutput = {ticketID: 'TICKET-1234', instanceID: 'https://something-1.atlassian.net'};
            expect(ref.current?.getIssueKey()).toEqual(expectedOutput);
        });

        test('should return the expected output when URL matches the second regex pattern', () => {
            const ref = React.createRef<TicketPopover>();
            render(
                <TicketPopover
                    {...mockProps1}
                    href='https://something-2.atlassian.net/jira/issues/?selectedIssue=TICKET-1234'
                    ref={ref}
                />,
            );
            const expectedOutput = {ticketID: 'TICKET-1234', instanceID: 'https://something-2.atlassian.net'};
            expect(ref.current?.getIssueKey()).toEqual(expectedOutput);
        });

        test('should return null when URL does not match any pattern', () => {
            const ref = React.createRef<TicketPopover>();
            render(
                <TicketPopover
                    {...mockProps1}
                    href='https://something-invalid.atlassian.net/not-a-ticket'
                    ref={ref}
                />,
            );
            expect(ref.current?.getIssueKey()).toEqual(null);
        });

        test('should return null when the URL does not contain the ticket ID', () => {
            const ref = React.createRef<TicketPopover>();
            render(
                <TicketPopover
                    {...mockProps1}
                    href='https://something-2.atlassian.net/jira/issues/?selectedIssue='
                    ref={ref}
                />,
            );
            expect(ref.current?.getIssueKey()).toEqual(null);
        });

        test('should return null when no instance is connected', () => {
            const ref = React.createRef<TicketPopover>();
            render(
                <TicketPopover
                    {...mockProps2}
                    href='https://something-2.atlassian.net/jira/issues/?selectedIssue='
                    ref={ref}
                />,
            );
            expect(ref.current?.getIssueKey()).toEqual(null);
        });
    });
});
