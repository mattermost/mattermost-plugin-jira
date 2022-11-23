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

const isAssigned = ' is assigned';
const unAssigned = 'Unassigned';
const jiraTicketTitleMaxLength = 80;

enum myStatus {
    INDETERMINATE = 'indeterminate',
    DONE = 'done',
}

const myStatusClasses: Record<string, string> = {
    [myStatus.INDETERMINATE]: ' ticket-status--indeterminate',
    [myStatus.DONE]: ' ticket-status--done',
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

        if (!ticketID && this.props.href.includes('atlassian.net/browse')) {
            ticketID = this.props.href.split('|')[0].split('?')[0].split('/browse/')[1];
        }

        this.state = {
            href: this.props.href,
            isLoaded: false,
            ticketId: ticketID,
        };
    }

    getIssueKey = () => {
        let ticketID = '';
        if (this.props.href.includes('selectedIssue')) {
            ticketID = this.props.href.split('selectedIssue=')[1].split('&')[0];
        }

        if (!ticketID && this.props.href.includes('atlassian.net/browse')) {
            ticketID = this.props.href.split('|')[0].split('?')[0].split('/browse/')[1];
        }

        return ticketID;
    }

    init() {
        let instanceID = this.props.defaultUserInstanceID || '';

        if (this.props.connectedInstances.length === 1) {
            instanceID = this.props.connectedInstances[0].instance_id;
        }

        const ticketID = this.getIssueKey();
        if (ticketID) {
            this.props.getIssueByKey(ticketID, instanceID);
        }
    }

    componentDidMount() {
        const {connected, ticketDetails, isLoaded: isPropsLoaded} = this.props;
        const {ticketId, isLoaded: isStateLoaded} = this.state;
        if (connected && !isStateLoaded && ((ticketDetails && ticketDetails.ticketId) !== ticketId || !isPropsLoaded)) {
            this.init();
        } else if (connected && !isStateLoaded && ticketDetails && ticketDetails.ticketId === ticketId) {
            this.setTicket(this.props);
        }
    }

    componentDidUpdate() {
        const {isLoaded: isPropsLoaded, ticketDetails} = this.props;
        const {ticketId, isLoaded: isStateLoaded} = this.state;
        if (isPropsLoaded && !isStateLoaded && ticketDetails && ticketDetails.ticketId === ticketId) {
            this.setTicket(this.props);
        }
    }

    setTicket(data: Props) {
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
        let ticketStatusClass = 'default-style';

        const myStatusClass = myStatusClasses[ticketStatus.toLowerCase()];
        if (myStatusClass) {
            ticketStatusClass += myStatusClass;
        } else {
            ticketStatusClass += ' ticket-status--default';
        }

        return <span className={ticketStatusClass}>{ticketStatus}</span>;
    }

    labelList(labels: string[]) {
        if (!labels.length) {
            return null;
        }

        let totalString = 0;
        let totalHide = 0;
        const labelList = labels.map((label: string, key: number): JSX.Element => {
            if (totalString < 3) {
                totalString++;
                return (
                    <span
                        key={key}
                        className='popover-labels__label-list'
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
                        className='popover-labels__label-list'
                    >
                        {moreLabels}
                    </span>);
            }

            return <></>;
        });
        return (<div className='popover-labels__label'>{labelList}</div>);
    }

    render() {
        if (!this.state.isLoaded) {
            return (<p/>);
        }

        const {ticketDetails, href: jiraTicketURI} = this.state;

        let jiraAvatar = null;
        let jiraIssueIconURI = null;
        let jiraTicketKey = null;
        let jiraTicketTitle = null;
        let jiraTicketAssigneeAvatarURI = null;
        let jiraTicketAssigneeName = null;
        let jiraTicketStatusName = null;
        let jiraTicketDescription = null;
        let jiraTicketVersions = null;
        let jiraTicketLabels = null;
        if (ticketDetails) {
            jiraAvatar = ticketDetails.jiraIcon;
            jiraIssueIconURI = ticketDetails.issueIcon;
            jiraTicketKey = ticketDetails.ticketId;
            jiraTicketTitle = ticketDetails.summary;
            jiraTicketAssigneeAvatarURI = ticketDetails.assigneeAvatar;
            jiraTicketAssigneeName = ticketDetails.assigneeName;
            jiraTicketStatusName = ticketDetails.statusKey;
            jiraTicketDescription = ticketDetails.description;
            jiraTicketVersions = ticketDetails.versions;
            jiraTicketLabels = ticketDetails.labels;
        }

        return (
            <div className='ticket-popover'>
                <div className='popover-header'>
                    <div className='popover-header__container'>
                        <a
                            href={jiraTicketURI}
                            title='Go to ticket'
                        >
                            <img
                                src={jiraAvatar}
                                width={14}
                                height={14}
                                alt='jira-avatar'
                                className='popover-header__avatar'
                            />
                        </a>
                        <a
                            href={jiraTicketURI}
                            className='popover-header__keyword'
                        >
                            <span className='jira-ticket-key'>{jiraTicketKey}</span>
                            <img
                                alt='jira-issue-icon'
                                width='14'
                                height='14'
                                src={jiraIssueIconURI}
                            />
                        </a>
                    </div>
                </div>
                <div className='popover-body'>
                    <div className='popover-body__title'>
                        <a href={jiraTicketURI}>
                            <h5>{this.truncateString(jiraTicketTitle as string, jiraTicketTitleMaxLength)}</h5>
                        </a>
                        {this.tagTicketStatus(jiraTicketStatusName as string)}
                    </div>
                    <div className='popover-body__description'>
                        {jiraTicketDescription}
                    </div>
                    <div className='popover-body__labels'>
                        {this.fixVersionLabel(jiraTicketVersions as string)}
                        {this.labelList(jiraTicketLabels as string[])}
                    </div>
                </div>
                <div className='popover-footer'>
                    {jiraTicketAssigneeAvatarURI ? (
                        <img
                            className='popover-footer__assigner-profile'
                            src={jiraTicketAssigneeAvatarURI}
                            alt='jira assigner profile'
                        />
                    ) : (
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
                    {jiraTicketAssigneeName ? (
                        <span>
                            <span className='popover-footer__assigner-name'>
                                {jiraTicketAssigneeName}
                            </span>
                            <span className='popover-footer__assigner--assigned'>
                                {isAssigned}
                            </span>
                        </span>
                    ) : (
                        <span className='popover-footer__assigner--assigned'>
                            {unAssigned}
                        </span>
                    )
                    }
                </div>
            </div>
        );
    }
}
