import React from 'react';
import {shallow} from 'enzyme';

import {Instance, InstanceType} from 'types/model';

import TicketPopover, {Props} from './jira_ticket_tooltip';

describe('components/jira_ticket_tooltip', () => {
    describe('getIssueKey', () => {
        const mockConnectedInstances: Instance[] = [
            {
                instance_id: '1',
                type: InstanceType.CLOUD,
            },
            {
                instance_id: '2',
                type: InstanceType.SERVER,
            },
        ];

        const props: Props = {
            href: '',
            show: false,
            connected: false,
            connectedInstances: mockConnectedInstances,
            fetchIssueByKey: jest.fn(),
        };

        test('should return the expected output when URL matches the first regex pattern', () => {
            const wrapper = shallow(
                <TicketPopover
                    {...props}
                    href='https://something.atlassian.net/browse/TICKET-1234'
                />
            );
            const instance = wrapper.instance() as TicketPopover;
            const expectedOutput = {ticketID: 'TICKET-1234', instanceID: '1'};
            expect(instance.getIssueKey()).toEqual(expectedOutput);
        });

        test('should return the expected output when URL matches the second regex pattern', () => {
            const wrapper = shallow(
                <TicketPopover
                    {...props}
                    href='https://something.atlassian.net/jira/issues/?selectedIssue=TICKET-1234'
                />
            );
            const instance = wrapper.instance() as TicketPopover;
            const expectedOutput = {ticketID: 'TICKET-1234', instanceID: '1'};
            expect(instance.getIssueKey()).toEqual(expectedOutput);
        });

        test('should return null when URL does not match any pattern', () => {
            const wrapper = shallow(
                <TicketPopover
                    {...props}
                    href='https://something.atlassian.net/not-a-ticket'
                />
            );
            const instance = wrapper.instance() as TicketPopover;
            expect(instance.getIssueKey()).toEqual(null);
        });
    });
});
