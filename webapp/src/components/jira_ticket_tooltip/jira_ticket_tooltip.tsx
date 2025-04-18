// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {ReactNode} from 'react';
import ReactMarkdown from 'react-markdown';

import {Instance} from 'types/model';
import SVGWrapper from 'components/svgWrapper';
import {SVGIcons} from 'components/plugin_constants/icons';
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

    fetchIssue = (show: boolean, connected: boolean, ticketId?: string, ticketDetails?: TicketDetails | null): void => {
        const issueKey = this.getIssueKey();
        if (!issueKey) {
            return;
        }

        if (!show) {
            return;
        }

        if (ticketId && !ticketDetails) {
            this.props.fetchIssueByKey(ticketId, issueKey.instanceID).then((res: {data?: TicketData, error?: any}) => {
                if (res.error) {
                    this.setState({error: 'There was a problem loading the details for this Jira link'});
                    return;
                }

                const updatedTicketDetails = getJiraTicketDetails(res.data);
                if (connected && updatedTicketDetails && updatedTicketDetails.ticketId === ticketId) {
                    this.setState({
                        ticketDetails: updatedTicketDetails,
                        error: null,
                    });
                }
            });
        }
    };

    componentDidMount(): void {
        this.fetchIssue(this.props.show, this.props.connected, this.state.ticketId, this.state.ticketDetails);
    }

    componentDidUpdate(): void {
        this.fetchIssue(this.props.show, this.props.connected, this.state.ticketId, this.state.ticketDetails);
    }

    fixVersionLabel(fixVersion: string): ReactNode {
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

    tagTicketStatus(ticketStatus: string): ReactNode {
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
                    <SVGWrapper
                        width={30}
                        height={30}
                        fill='#FF0000'
                        className='bi bi-exclamation-triangle'
                    >
                        {SVGIcons.exclamationTriangle}
                    </SVGWrapper>
                    <div className='jira-issue-error-message'>{error}</div>
                    <p className='jira-issue-error-footer'>{'Check your connection or try again later'}</p>
                </div>
            );
        }

        // Format the ticket summary by trimming spaces, replacing multiple spaces with one, truncating to `jiraTicketSummaryMaxLength`, and adding '...' if it exceeds the limit.
        const formattedSummary = ticketDetails?.summary ? `${ticketDetails.summary.trim().split(/\s+/).join(' ')
            .substring(0, jiraTicketSummaryMaxLength)}${ticketDetails.summary.trim().split(/\s+/).join(' ').length > jiraTicketSummaryMaxLength ? '...' : ''}` : '';

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
                            title={ticketDetails?.summary}
                            rel='noopener noreferrer'
                        >
                            <h5 className='tooltip-ticket-summary'>{ticketDetails.summary && ticketDetails.summary.substring(0, jiraTicketSummaryMaxLength)}</h5>
                        </a>
                        {this.tagTicketStatus(ticketDetails.statusKey)}
                    </div>
                    <div className='popover-body__description'>
                        <ReactMarkdown>{ticketDetails.description && `${ticketDetails.description.substring(0, maxTicketDescriptionLength).trim()}${ticketDetails.description.length > maxTicketDescriptionLength ? '...' : ''}`}</ReactMarkdown>
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
