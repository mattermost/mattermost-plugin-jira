import React from 'react';
import './ticketStyle.scss';

import {Dispatch} from 'redux';

import {Instance} from 'types/model';

export type Props = {
    href: string;
    show: boolean;
    connected: boolean;
    isLoaded?: boolean;
    ticketDetails?: TicketDetails;
    defaultUserInstanceID?: string;
    connectedInstances: Instance[];
    getIssueByKey: (issueKey: string, instanceID: string) => (dispatch: Dispatch, getState: any) => Promise<{
        data?: TicketData;
        error?: any;
    }>;
    setTicket?: (ticketDetails: {}) => void;
}

export type State = {
    href: string;
    isLoaded: boolean;
    ticketId: string;
    ticketDetails?: TicketDetails;
};

export default class TicketPopover extends React.PureComponent<Props, State> {
    truncateString(str: string, num: number) {
        if (num > str.length) {
            return str;
        }
        return `${str.substring(0, num)}...`;
    }

    constructor(Props: Props) {
        super(Props);
        let ticketID = '';
        if (this.props.href.includes('selectedIssue')) {
            ticketID = this.props.href.split('selectedIssue=')[1].split('&')[0];
        }

        if (ticketID === '' && this.props.href.includes('atlassian.net/browse')) {
            ticketID = this.props.href.split('|')[0].split('?')[0].split('/browse/')[1];
        }

        this.state = {
            href: this.props.href,
            isLoaded: false,
            ticketId: ticketID,
        };
    }

    async init() {
        let instanceID = '';
        if (this.props.connectedInstances.length === 1) {
            instanceID = this.props.connectedInstances[0].instance_id;
        } else if (this.props.defaultUserInstanceID) {
            instanceID = this.props.defaultUserInstanceID;
        }

        let ticketID = '';
        if (this.props.href.includes('selectedIssue')) {
            ticketID = this.props.href.split('selectedIssue=')[1].split('&')[0];
            this.props.getIssueByKey(ticketID, instanceID);
        }

        if (ticketID === '' && this.props.href.includes('atlassian.net/browse')) {
            ticketID = this.props.href.split('|')[0].split('?')[0].split('/browse/')[1];
            if (ticketID) {
                this.props.getIssueByKey(ticketID, instanceID);
            }
        }
    }

    componentDidMount() {
        if (this.props.connected && !this.state.isLoaded && ((this.props.ticketDetails && this.props.ticketDetails.ticketId !== this.state.ticketId) || !this.props.isLoaded)) {
            this.init();
        } else if (this.props.connected && !this.state.isLoaded && this.props.ticketDetails && this.props.ticketDetails.ticketId === this.state.ticketId) {
            this.setTicket(this.props);
        }
    }

    componentDidUpdate(): void {
        if (this.props.isLoaded && !this.state.isLoaded && this.props.ticketDetails && this.props.ticketDetails.ticketId === this.state.ticketId) {
            this.setTicket(this.props);
        }
    }

    setTicket(data: Props): void{
        this.setState({
            isLoaded: true,
            ticketDetails: data.ticketDetails,
        });
    }

    fixVersionLabel(fixVersion: string) {
        if (fixVersion) {
            const fixVersionString = 'Fix Version :';
            return (
                <div className='fix-version-label'>
                    {fixVersionString}
                    <span className='fix-version-label-value'>
                        {fixVersion}
                    </span>
                </div>
            );
        }
        return (<span/>);
    }

    tagTicketStatus(ticketStatus: string) {
        if (ticketStatus.toLowerCase() === 'indeterminate') {
            return (<span className='ticket-status-indeterminate default-style'>{ticketStatus}</span>);
        }

        if (ticketStatus.toLowerCase() === 'done') {
            return (<span className='ticket-status-done default-style'>{ticketStatus}</span>);
        }

        return (<span className='ticket-status-default default-style'>{ticketStatus}</span>);
    }

    labelList(labels: string[]) {
        if (labels.length) {
            let totalString = 0;
            let totalHide = 0;
            const labelList = labels.map((label: string, key: number): JSX.Element => {
                if (totalString < 3) {
                    totalString++;
                    return (
                        <span
                            key={key}
                            className='jiraticket-popover-label-list'
                        >
                            {label}
                        </span>);
                }
                totalHide++;
                if (key === labels.length - 1) {
                    const moreLabels = `+${totalHide}more`;
                    return (
                        <span
                            key={key}
                            className='jiraticket-popover-label-list'
                        >
                            {moreLabels}
                        </span>);
                }
                return <></>;
            });

            return (<div className={'ticket-popover-label'}>{labelList}</div>);
        }
        return null;
    }

    render() {
        if (!this.state.isLoaded) {
            return (<p/>);
        }
        const ticketDetails = this.state.ticketDetails;
        const jiraTicketURI = this.state.href;
        const jiraAvatar = ticketDetails && ticketDetails.jiraIcon;
        const jiraIssueIconURI = ticketDetails && ticketDetails.issueIcon;
        const jiraTicketKey = ticketDetails && ticketDetails.ticketId;
        const jiraTicketTitle = ticketDetails && ticketDetails.summary;
        const jiraTicketAssigneeAvatarURI = ticketDetails && ticketDetails.assigneeAvatar;
        const jiraTicketAssigneeName = ticketDetails && ticketDetails.assigneeName;
        const jiraTicketStatusName = ticketDetails && ticketDetails.statusKey;
        const jiraTicketDescription = ticketDetails && ticketDetails.description;
        const jiraTicketVersions = ticketDetails && ticketDetails.versions;
        const jiraTicketLabels = ticketDetails && ticketDetails.labels;
        const isAssigned = ' is assigned';
        const unAssigned = 'Unassigned';

        return (
            <div className={'ticket-popover'}>
                <div className={'ticket-popover-header'}>
                    <div className={'ticket-popover-header-container'}>
                        <a
                            href={jiraTicketURI}
                            title={'goto ticket'}
                        >
                            <img
                                src={jiraAvatar}
                                width={14}
                                height={14}
                                alt={'jira-avatar'}
                                className={'ticket-popover-header-avatar'}
                            />
                        </a>
                        <a
                            href={jiraTicketKey}
                            className={'ticket-popover-keyword'}
                        >
                            <span className='jira-ticket-key'>{jiraTicketKey}</span>
                            <img
                                alt={'jira-issue-icon'}
                                width='14'
                                height='14'
                                src={jiraIssueIconURI}
                            />
                        </a>
                    </div>
                </div>
                <div className={'ticket-popover-body'}>
                    <div className={'ticket-popover-title'}>
                        <a href={jiraTicketURI}>
                            <h5>{this.truncateString(jiraTicketTitle, 80)}</h5>
                        </a>
                        {this.tagTicketStatus(jiraTicketStatusName)}
                    </div>
                    <div className={'ticket-popover-description'}>
                        {jiraTicketDescription}
                    </div>
                    <div className={'ticket-popover-labels'}>
                        {this.fixVersionLabel(jiraTicketVersions)}
                        {this.labelList(jiraTicketLabels)}
                    </div>
                </div>
                <div className={'ticket-popover-footer'}>
                    {jiraTicketAssigneeAvatarURI ?
                        (
                            <img
                                className={'ticket-popover-footer-assigner-profile'}
                                src={jiraTicketAssigneeAvatarURI}
                                alt={'jira assigner profile'}
                            />
                        ) :
                        (
                            <span className='default-avatar'>
                                <svg
                                    width='18'
                                    height='18'
                                    viewBox='0 0 18 18'
                                    role='presentation'
                                >
                                    <g
                                        fill='white'
                                        fillRule='evenodd'
                                    >
                                        <path
                                            d='M3.5 14c0-1.105.902-2 2.009-2h7.982c1.11 0 2.009.894 2.009 2.006v4.44c0 3.405-12 3.405-12 0V14z'
                                        />
                                        <circle
                                            cx='9'
                                            cy='6'
                                            r='3.5'
                                        />
                                    </g>
                                </svg>
                            </span>
                        )
                    }
                    {jiraTicketAssigneeName ?
                        (
                            <span>
                                <span className={'ticket-popover-footer-assigner-name'}>
                                    {jiraTicketAssigneeName}
                                </span>
                                <span className={'ticket-popover-footer-assigner-is-assigned'}>
                                    {isAssigned}
                                </span>
                            </span>
                        ) :
                        (
                            <span className={'ticket-popover-footer-assigner-is-assigned'}>
                                {unAssigned}
                            </span>
                        )
                    }
                </div>
            </div>
        );
    }
}
