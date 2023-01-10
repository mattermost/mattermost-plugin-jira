import React, {ReactNode} from 'react';
import './ticketStyle.scss';

import {Dispatch} from 'redux';

import {Instance} from 'types/model';
import DefaultAvatar from 'components/default_avatar/default_avatar';

export type Props = {
    href: string;
    show: boolean;
    connected: boolean;
    ticketDetails?: TicketDetails;
    connectedInstances: Instance[];
    fetchIssueByKey: (issueKey: string, instanceID: string) => (dispatch: Dispatch, getState: any) => Promise<{
        data?: TicketData;
    }>;
}

export type State = {
    isLoaded: boolean;
    ticketId: string;
    ticketDetails?: TicketDetails;
};

const isAssignedLabel = ' is assigned';
const unAssignedLabel = 'Unassigned';
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
    constructor(props: Props) {
        super(props);
        const {ticketID} = this.getIssueKey();

        this.state = {
            isLoaded: false,
            ticketId: ticketID,
        };
    }

    getIssueKey = () => {
        let ticketID = '';
        let instanceID = '';

        for (const instance of this.props.connectedInstances) {
            instanceID = instance.instance_id;

            try {
                if (this.props.href.includes(instanceID)) {
                    if (this.props.href.includes('selectedIssue=')) {
                        ticketID = this.props.href.split('selectedIssue=')[1].split('&')[0];
                        break;
                    }

                    ticketID = this.props.href.split('?')[0].split('/browse/')[1];
                    break;
                }
            } catch {
                ticketID = '';
            }
        }

        return {ticketID, instanceID};
    }

    init() {
        const {ticketID, instanceID} = this.getIssueKey();
        if (ticketID) {
            this.props.fetchIssueByKey(ticketID, instanceID);
        }
    }

    isUserConnected() {
        const {connected} = this.props;
        const {isLoaded} = this.state;

        return connected && !isLoaded;
    }

    componentDidMount() {
        const {ticketDetails} = this.props;
        const {ticketId} = this.state;
        if (this.isUserConnected() && ((ticketDetails && ticketDetails.ticketId) !== ticketId)) {
            this.init();
        } else if (this.isUserConnected() && ticketDetails && ticketDetails.ticketId === ticketId) {
            this.setTicket(this.props);
        }
    }

    componentDidUpdate() {
        const {ticketDetails} = this.props;
        const {ticketId, isLoaded: isStateLoaded} = this.state;
        if (!isStateLoaded && ticketDetails && ticketDetails.ticketId === ticketId) {
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

        return null;
    }

    tagTicketStatus(ticketStatus: string) {
        let ticketStatusClass = 'default-style ticket-status--default';

        const myStatusClass = myStatusClasses[ticketStatus.toLowerCase()];
        if (myStatusClass) {
            ticketStatusClass = 'default-style ' + myStatusClass;
        }

        return <span className={ticketStatusClass}>{ticketStatus}</span>;
    }

    renderLabelList(labels: string[]) {
        if (!labels.length) {
            return null;
        }

        return (
            <div className='popover-labels__label'>
                {
                    labels.map((label: string, key: number): ReactNode => {
                        if (key < 3 || (key === labels.length - 1 && labels.length - 3 > 0)) {
                            return (
                                <span
                                    key={key}
                                    className='popover-labels__label-list'
                                >
                                    {key === labels.length - 1 && labels.length - 3 > 0 ? `+${labels.length - 3} more` : label}
                                </span>);
                        }

                        return null;
                    })
                }
            </div>
        );
    }

    render() {
        if (!this.state.isLoaded) {
            return (<p/>);
        }

        const {ticketDetails} = this.state;

        return (
            <div className='jira-issue-tooltip'>
                <div className='popover-header'>
                    <div className='popover-header__container'>
                        <a
                            href={this.props.href}
                            title='Go to ticket'
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            <img
                                src={ticketDetails && ticketDetails.jiraIcon}
                                width={14}
                                height={14}
                                alt='jira-avatar'
                                className='popover-header__avatar'
                            />
                        </a>
                        <a
                            href={this.props.href}
                            className='popover-header__keyword'
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            <span className='jira-ticket-key'>{ticketDetails && ticketDetails.ticketId}</span>
                            <img
                                alt='jira-issue-icon'
                                width='14'
                                height='14'
                                src={ticketDetails && ticketDetails.issueIcon}
                            />
                        </a>
                    </div>
                </div>
                <div className='popover-body'>
                    <div className='popover-body__title'>
                        <a
                            href={this.props.href}
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            <h5>{ticketDetails && ticketDetails.summary.substring(0, jiraTicketTitleMaxLength)}</h5>
                        </a>
                        {ticketDetails && this.tagTicketStatus(ticketDetails.statusKey)}
                    </div>
                    <div className='popover-body__description'>
                        {ticketDetails && ticketDetails.description}
                    </div>
                    <div className='popover-body__labels'>
                        {ticketDetails && this.fixVersionLabel(ticketDetails.versions)}
                        {ticketDetails && this.renderLabelList(ticketDetails.labels)}
                    </div>
                </div>
                <div className='popover-footer'>
                    {ticketDetails && ticketDetails.assigneeAvatar ? (
                        <img
                            className='popover-footer__assignee-profile'
                            src={ticketDetails.assigneeAvatar}
                            alt='jira assignee profile'
                        />
                    ) : <DefaultAvatar/>
                    }
                    {ticketDetails && ticketDetails.assigneeName ? (
                        <span>
                            <span className='popover-footer__assignee-name'>
                                {ticketDetails.assigneeName}
                            </span>
                            <span className='popover-footer__assignee--assigned'>
                                {isAssignedLabel}
                            </span>
                        </span>
                    ) : (
                        <span className='popover-footer__assignee--assigned'>
                            {unAssignedLabel}
                        </span>
                    )
                    }
                </div>
            </div>
        );
    }
}
