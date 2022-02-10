import React, {Fragment, PureComponent} from 'react';
import './ticketStyle.scss';

import {Instance, GetConnectedResponse} from 'types/model';

export type Props = {
    href: string;
    connected: boolean;
    isloaded: string;
    assigneeName: string;
    assigneeAvatar: string;
    labels: any[];
    versions: string;
    description: string;
    summary: string;
    ticketId: string;
    jiraIcon: string;
    statusKey: string;
    issueIcon: string;
    defaultUserInstanceID: string;
    installedInstances: Instance[];
    connectedInstances: Instance[];
    getIssueByKey: (ticketId: string, instanceID: string) => void;
    getConnected: () => Promise<GetConnectedResponse>;
}
export default class TicketPopover extends React.PureComponent<Props> {
    truncateString(str: string, num: number) {
        if (num > str.length) {
            return str;
        }
        return `${str.substring(0, num)}...`;
    }

    async init() {
        let instanceID = '';
        if (this.props.connectedInstances.length === 1) {
            instanceID = this.props.connectedInstances[0].instance_id;
        } else if (this.props.defaultUserInstanceID) {
            instanceID = this.props.defaultUserInstanceID;
        }

        if (this.props.href.includes('atlassian.net/browse')) {
            this.props.getIssueByKey(this.props.href.split('|')[0].split('/browse/')[1], instanceID);
        } else if (this.props.href.includes('atlassian.net/jira/software')) {
            const urlParams = new URLSearchParams(this.props.href);
            const selectedIssue = urlParams.get('selectedIssue');
            if (selectedIssue != null) {
                this.props.getIssueByKey(selectedIssue, instanceID);
            }
        }
    }

    componentDidMount() {
        this.props.getConnected();
        if (this.props.connected && !this.props.isloaded) {
            this.init();
        }
    }

    fixVersionLabel(fixVersion: string) {
        if (fixVersion) {
            const fixVersionString = 'Fix Version :';
            return (
                <div className='fix-version-label'
                    style={{color: '#333', margin: '16px 0px', textAlign: 'left', fontFamily: 'open sans', fontSize: '10px', padding: '0px 0px 2px 0px'}}>
                    {fixVersionString}
                    <span 
                        className='fix-version-label-value' style={{backgroundColor: 'rgba(63, 67, 80, 0.08)', padding: '1px 8px', fontWeight: 600, borderRadius: '2px'}}>
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
            marginTop: '4px',
            padding: '0px 8px',
            align: 'center',
            height: 20,
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
            let labelList = labels.map((label: any, key: any) => {
                if (totalString >= 45 || totalString + label.length >= 45) {
                    totalHide++;
                    return null;
                }
                totalString += label.length + 3;
                return <span key={key} 
                    className='jiraticket-popover-label-list'>{label}</span>;
            });
            const moreLabels = `+${totalHide}more`;
            return (
                <Fragment>
                    <div className={'ticket-popover-label'}>
                        {labelList}
                    </div>
                    {
                        totalHide !== 0 ? (<div className={'jiraticket-popover-total-hide-label'}> moreLabels</div>) : null
                    }

                </Fragment>
            );
        }
        return (<span/>);
    }

    render() {
        if (!this.props.isloaded) {
            return (<p/>);
        }
        const jiraTicketURI = this.props.href;
        const jiraAvatar = this.props.jiraIcon;
        const jiraIssueIconURI = this.props.issueIcon;
        const jiraTicketKey = this.props.ticketId;
        const jiraTicketTitle = this.props.summary;
        const jiraTicketAssigneeAvatarURI = this.props.assigneeAvatar;
        const jiraTicketAssigneeName = this.props.assigneeName;
        const jiraTicketStatusName = this.props.statusKey;
        const jiraTicketDescription = this.props.description;
        const jiraTicketVersions = this.props.versions;
        const jiraTicketLabels = this.props.labels;
        const isAssigned = 'is assigned';
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
                    <img
                        className={'ticket-popover-footer-assigner-profile'}
                        src={jiraTicketAssigneeAvatarURI}
                        alt={'jira assigner profile'}
                    />
                    <span className={'ticket-popover-footer-assigner-name'}>
                        {jiraTicketAssigneeName}
                    </span>
                    <span className={'ticket-popover-footer-assigner-is-assigned'}>
                        {isAssigned}
                    </span>
                </div>
            </div>
        );
    }
}
