import React, {ReactNode} from 'react';

import {Instance} from 'types/model';
import {TicketData, TicketDetails} from 'types/tooltip';
import DefaultAvatar from 'components/default_avatar/default_avatar';

import './ticketStyle.scss';
import {getJiraTicketDetails} from 'utils/jira_issue_metadata';

export type Props = {
    href: string;
    show: boolean;
    connected: boolean;
    connectedInstances: Instance[];
    fetchIssueByKey: (issueKey: string, instanceID: string) => Promise<{data?: TicketData}>;
}

export type State = {
    ticketId: string;
    ticketDetails?: TicketDetails | null;
    error: string | null;
};

const isAssignedLabel = ' is assigned';
const unAssignedLabel = 'Unassigned';
const jiraTicketSummaryMaxLength = 80;
const maxTicketDescriptionLength = 160;

enum myStatus {
    INDETERMINATE = 'indeterminate',
    DONE = 'done',
}

const myStatusClasses: Record<string, string> = {
    [myStatus.INDETERMINATE]: 'ticket-status--indeterminate',
    [myStatus.DONE]: 'ticket-status--done',
};

export default class TicketPopover extends React.PureComponent<Props, State> {
    constructor(props: Props) {
        super(props);
        const issueKey = this.getIssueKey();
        let ticketID = '';
        if (issueKey) {
            ticketID = issueKey.ticketID;
        }

        this.state = {
            ticketId: ticketID,
            error: null,
        };
    }

    getIssueKey = () => {
        let ticketID = '';
        let instanceID = '';

        for (const instance of this.props.connectedInstances) {
            instanceID = instance.instance_id;

            if (!this.props.href.includes(instanceID)) {
                continue;
            }

            // We already check href.includes above in the if statement before this try block
            try {
                const regex = /(https|http):\/\/.*\/.*\?.*selectedIssue=([\w-]+)&?.*|(https|http):\/\/.*\/browse\/([\w-]+)?.*/;
                const result = regex.exec(this.props.href);
                if (result) {
                    ticketID = result[2] || result[4];
                    return {ticketID, instanceID};
                }
                break;
            } catch (e) {
                break;
            }
        }

        return null;
    };

    componentDidUpdate() {
        const issueKey = this.getIssueKey();
        if (!issueKey) {
            return;
        }

        const {instanceID} = issueKey;
        const {ticketId, ticketDetails} = this.state;
        if (!ticketDetails && this.props.show && ticketId) {
            this.props.fetchIssueByKey(this.state.ticketId, instanceID).then((res: { data?: TicketData; error?: any}) => {
                if (res.error) {
                    this.setState({error: 'There was a problem loading the details for this Jira link'});
                    return;
                }
                const updatedTicketDetails = getJiraTicketDetails(res.data);
                if (this.props.connected && updatedTicketDetails && updatedTicketDetails.ticketId === ticketId) {
                    this.setState({
                        ticketDetails: updatedTicketDetails,
                        error: null,
                    });
                }
            });
        }
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

        const myStatusClass = myStatusClasses[ticketStatus && ticketStatus.toLowerCase()];
        if (myStatusClass) {
            ticketStatusClass = 'default-style ' + myStatusClass;
        }

        return <span className={ticketStatusClass}>{ticketStatus}</span>;
    }

    renderLabelList(labels: string[]) {
        if (!labels || !labels.length) {
            return null;
        }

        return (
            <div className='popover-labels__label'>
                {
                    labels.map((label: string, key: number): ReactNode => {
                        // Return an element for the first three labels and if there are more than three labels, then return a combined label for the remaining labels
                        if (key < 3) {
                            return (
                                <span
                                    key={key}
                                    className='popover-labels__label-list'
                                >
                                    {label}
                                </span>);
                        }

                        if (key === labels.length - 1 && labels.length > 3) {
                            return (
                                <span
                                    key={key}
                                    className='popover-labels__label-list'
                                >
                                    {`+${labels.length - 3} more`}
                                </span>);
                        }

                        return null;
                    })
                }
            </div>
        );
    }

    render() {
        if (!this.state.ticketId || (!this.state.ticketDetails && !this.props.show)) {
            return null;
        }

        const {ticketDetails, error} = this.state;
        if (error) {
            return (
                <div className='jira-issue-tooltip jira-issue-tooltip-error'>
                    <span
                        className='jira-issue-error-icon fa fa-exclamation-triangle'
                        style={{color: 'red'}}
                        title={'Hazard Icon'}
                    />
                    <div className='jira-issue-error-message'>{error}</div>
                    <p className='jira-issue-error-footer'>{'Check your connection or try again later'}</p>
                </div>
            );
        }

        if (!ticketDetails) {
            // Display the spinner loader while ticket details are being fetched
            return (
                <div className='jira-issue-tooltip jira-issue-tooltip-loading'>
                    <span
                        className='jira-issue-spinner fa fa-spin fa-spinner'
                        title={'Loading Icon'}
                    />
                </div>
            );
        }

        return (
            <div className='jira-issue-tooltip'>
                <div className='popover-header'>
                    <div className='popover-header__container'>
                        <a
                            href={this.props.href}
                            className='popover-header__keyword'
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            <span className='jira-ticket-key'>{ticketDetails.ticketId}</span>
                            <img
                                alt='jira-issue-icon'
                                width='14'
                                height='14'
                                src={ticketDetails.issueIcon}
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
                            <h5>{ticketDetails.summary && ticketDetails.summary.substring(0, jiraTicketSummaryMaxLength)}</h5>
                        </a>
                        {this.tagTicketStatus(ticketDetails.statusKey)}
                    </div>
                    <div className='popover-body__description'>
                        {ticketDetails.description && `${ticketDetails.description.substring(0, maxTicketDescriptionLength).trim()}${ticketDetails.description.length > maxTicketDescriptionLength ? '...' : ''}`}
                    </div>
                    <div className='popover-body__see-more-link'>
                        <a
                            href={this.props.href}
                            target='_blank'
                            rel='noopener noreferrer'
                        >
                            {'See more'}
                        </a>
                    </div>
                    <div className='popover-body__labels'>
                        {this.fixVersionLabel(ticketDetails.versions)}
                        {this.renderLabelList(ticketDetails.labels)}
                    </div>
                </div>
                <div className='popover-footer'>
                    {ticketDetails.assigneeAvatar ? (
                        <img
                            className='popover-footer__assignee-profile'
                            src={ticketDetails.assigneeAvatar}
                            alt='jira assignee profile'
                        />
                    ) : <DefaultAvatar/>
                    }
                    {ticketDetails.assigneeName ? (
                        <span>
                            <span className='popover-footer__assignee-name'>
                                {ticketDetails.assigneeName}
                            </span>
                            <span>
                                {isAssignedLabel}
                            </span>
                        </span>
                    ) : (
                        <span>
                            {unAssignedLabel}
                        </span>
                    )
                    }
                </div>
            </div>
        );
    }
}
