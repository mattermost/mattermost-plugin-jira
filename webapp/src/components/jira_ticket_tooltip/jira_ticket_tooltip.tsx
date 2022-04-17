import React, {Fragment, PureComponent} from 'react';
import './ticketStyle.scss';

import {Instance, GetConnectedResponse} from 'types/model';

export type Props = {
    href: string;
    show: boolean;
    connected: boolean;
    isloaded?: boolean;
    ticketDetails: any;
    defaultUserInstanceID?: string;
    connectedInstances: Instance[];
    getIssueByKey: (ticketId: string, instanceID: string) => Promise<{data: any; error?: Error}>;
    setTicket?: (ticketDetails: {}) => void;
}
export default class TicketPopover extends React.PureComponent<Props> {
    truncateString(str: string, num: number) {
        if (num > str.length) {
            return str;
        }
        return `${str.substring(0, num)}...`;
    }

    constructor(Props) {
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
            isloaded: false,
            assigneeName: '',
            assigneeAvatar: '',
            labels: '',
            versions: '',
            description: '',
            summary: '',
            ticketId: ticketID,
            jiraIcon: '',
            statusKey: '',
            issueIcon: '',
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
            this.props.getIssueByKey(ticketID, instanceID).then(({data, error}) => {
                if (error) {
                    return;
                }

                this.setTicket(data);
            });
        }

        if (ticketID === '' && this.props.href.includes('atlassian.net/browse')) {
            this.props.getIssueByKey(this.props.href.split('|')[0].split('?')[0].split('/browse/')[1], instanceID).then(({data, error}) => {
                if (error) {
                    return;
                }

                this.setTicket(data);
            });
        }
    }

    componentDidUpdate(prevProps: Props): void {
        if (!this.props.connected) {
            return;
        }

        if (!(this.props.show && !prevProps.show)) {
            return;
        }

        if (this.state.isLoaded) {
            return;
        }

        this.init();
    }

    setTicket(data: any): void{
        this.setState({
            isloaded: true,
            ticketDetails: data,
        });
    }

    fixVersionLabel(fixVersion: string) {
        if (fixVersion) {
            const fixVersionString = 'Fix Version :';
            return (
                <div
                    className='fix-version-label'
                    style={{color: '#333', margin: '16px 0px', textAlign: 'left', fontFamily: 'open sans', fontSize: '10px', padding: '0px 0px 2px 0px'}}
                >
                    {fixVersionString}
                    <span
                        className='fix-version-label-value'
                        style={{backgroundColor: 'rgba(63, 67, 80, 0.08)', padding: '1px 8px', fontWeight: 600, borderRadius: '2px'}}
                    >
                        {fixVersion}
                    </span>
                </div>
            );
        }
        return (<span/>);
    }

    tagTicketStatus(ticketStatus: string) {
        const defaultStyle = {
            fontFamily: 'open sans',
            fontStyle: 'normal',
            fontWeight: 600,
            fontSize: '12px',
            marginTop: '10px',
            padding: '4px 8px',
            align: 'center',
            marginBottom: '8px',
            borderRadius: '4px',
        };

        if (ticketStatus.toLowerCase() === 'indeterminate') {
            return (<span style={{...defaultStyle, color: '#FFFFFF', backgroundColor: '#1C58D9', borderRadius: '2px'}}>{ticketStatus}</span>);
        }

        if (ticketStatus.toLowerCase() === 'done') {
            return (<span style={{...defaultStyle, color: '#FFFFFF', backgroundColor: '#3DB887'}}>{ticketStatus}</span>);
        }

        return (<span style={{...defaultStyle, color: '#3F4350', backgroundColor: 'rgba(63, 67, 80, 0.16)'}}>{ticketStatus}</span>);
    }

    labelList(labels: string[]) {
        if (labels.length > 0) {
            let totalString = 0;
            let totalHide = 0;
            const labelList = labels.map((label: any, key: any): any => {
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
                return null;
            });

            return (<div className={'ticket-popover-label'}>{labelList}</div>);
        }
        return null;
    }

    render() {
        if (!this.state.isloaded) {
            return (<p/>);
        }
        const ticketDetails = this.state.ticketDetails;
        const assignee = ticketDetails && ticketDetails.fields && ticketDetails.fields.assignee ? ticketDetails.fields.assignee : null;
        const jiraTicketURI = this.state.href;
        const jiraAvatar = ticketDetails.fields.project.avatarUrls['48x48'];
        const jiraIssueIconURI = ticketDetails.fields.issuetype.iconUrl;
        const jiraTicketKey = ticketDetails.key;
        const jiraTicketTitle = ticketDetails.fields.summary;
        const jiraTicketAssigneeAvatarURI = assignee && assignee.avatarUrls && assignee.avatarUrls['48x48'] ? assignee.avatarUrls['48x48'] : '';
        const jiraTicketAssigneeName = assignee && assignee.displayName ? assignee.displayName : '';
        const jiraTicketStatusName = ticketDetails.fields.status.name;
        const jiraTicketDescription = ticketDetails.fields.description;
        const jiraTicketVersions = ticketDetails.fields.versions.lenght > 0 ? ticketDetails.fields.versions.versions[0] : '';
        const jiraTicketLabels = ticketDetails.fields.labels;
        const isAssigned = ' is assigned';
        const unAssigned = 'Unassigned';
        const Tippy = window.Tippy.default;

        const Tooltip = ({children, ...rest}) => <Tippy {...rest}>{children}</Tippy>;
        Tooltip.defaultProps = {
            animation: 'fade',
            arrow: true,
        };

        return (
            <Tippy >
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
                                <span style={{fontSize: 12}}>{jiraTicketKey}</span>
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
                        { jiraTicketAssigneeAvatarURI === '' ?
                            (
                                <span style={{backgroundColor: 'slategrey', borderRadius: '50%', marginRight: '5px', padding: '1px'}}>
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
                            ) :
                            (
                                <img
                                    className={'ticket-popover-footer-assigner-profile'}
                                    src={jiraTicketAssigneeAvatarURI}
                                    alt={'jira assigner profile'}
                                />
                            )
                        }
                        { jiraTicketAssigneeName === '' ?
                            (
                                <span className={'ticket-popover-footer-assigner-is-assigned'}>
                                    {unAssigned}
                                </span>
                            ) :
                            (
                                <span>
                                    <span className={'ticket-popover-footer-assigner-name'}>
                                        {jiraTicketAssigneeName}
                                    </span>
                                    <span className={'ticket-popover-footer-assigner-is-assigned'}>
                                        {isAssigned}
                                    </span>
                                </span>
                            )
                        }
                    </div>
                </div>
            </Tippy>
        );
    }
}
