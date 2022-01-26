import React, {Fragment, PureComponent} from 'react';
import './ticketStyle.scss';

import PropTypes from 'prop-types';

export default class TicketPopover extends PureComponent {
    static propTypes = {
        href: PropTypes.string,
        connected: PropTypes.any,
    }

    truncateString(str, num) {
        if (num > str.length) {
            return str;
        }
        return str.substring(0, num) + '...';
    }

    async init() {
        let ticketId = '';
        if (this.href.includes('attlasian.net/browse')) {
            ticketId = href.split('/browse/')[1].split('/');
            this.props.value = await getTicket(ticketId);
        }

        if (this.href.includes("atlassian.net/jira/software")) {
            const urlParams = new URLSearchParams(fi);
            ticketId = urlParams.get('selectedIssue');
            this.props.value = await getTicket(ticketId);
        }
    }

    componentDidMount() {
        if (this.connected) {
            this.init();
        }
    }

    //
    // eslint-disable-next-line consistent-return
    fixVersionLabel(fixVersion) {
        if (fixVersion) {
            return (<div
                className='fix-version-label'
                style={{
                    color: '#333',
                    margin: '16px 0px',
                    textAlign: 'left',
                    fontFamily: 'open sans',
                    fontSize: '10px',
                    padding: '0px 0px 2px 0px',
                }}>Fix Version: <span className="fix-version-label-value" style={{
                backgroundColor: 'rgba(63, 67, 80, 0.08)',
                padding: '1px 8px',
                fontWeight: '600',
                borderRadius: '2px',
            }}>{fixVersion}
            </span></div>);
        }
    }

    tagTicketStatus(ticketStatus, ticketStatusKey) {
        const defaultStyle = {
            fontFamily: 'open sans',
            fontStyle: 'normal',
            fontWeight: '600',
            fontSize: '12px',
            marginTop: '4px',
            padding: '0px 8px',
            align: 'center',
            height: 20,
            marginBottom: '8px',
            borderRadius: '4px',
        };
        if (ticketStatusKey === 'indeterminate') {
            return <span style={{
                ...defaultStyle,
                color: '#FFFFFF',
                backgroundColor: '#1C58D9',
                borderRadius: '2px',
            }}>{ticketStatus}</span>
        }

        if (ticketStatusKey === "done") {
            return <span style={{
                ...defaultStyle,
                color: '#FFFFFF',
                backgroundColor: '#3DB887',

            }}>{ticketStatus}</span>
        }

        // ticketStatus == "new" or other
        return <span style={{
            ...defaultStyle,
            color: '#3F4350',
            backgroundColor: 'rgba(63, 67, 80, 0.16)',
        }}>{ticketStatus}</span>
    }

    labelList(labels) {
        if (labels !== undefined && labels !== null) {
            let totalString = 0
            let totalHide = 0;
            return (
                <Fragment>
                    <div className={'ticket-popover-label'}>
                        {labels.map(function (label) {
                            if (totalString >= 45 || totalString + label.length >= 45) {
                                totalHide++;
                                return null;
                            }
                            totalString += label.length + 3;
                            return <span className="jiraticket-popover-label-list">{label}</span>;
                        })}
                    </div>
                    {
                        totalHide !== 0 ? (
                            <div className={'jiraticket-popover-total-hide-label'}>+{totalHide}more</div>) : null
                    }

                </Fragment>
            )
        }
    }


    render() {
        const jiraTicketURI = 'www.facebook.com';//this.props.value.self
        const jiraAvatar = 'https://icons.iconarchive.com/icons/hopstarter/superhero-avatar/128/Avengers-Hulk-icon.png'; //
        const jiraStatusIconURI = 'https://digitalcenter.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10315?size=medium';//this.props.value.fields.status.iconUrl
        const jiraTicketKey = 'MM-37566';//this.props.value.key
        const jiraTicketTitle = 'RN: Mobile V2: In-App Notifications';//this.props.value.names
        const jiraTicketAssigneeAvatarURI = 'https://icons.iconarchive.com/icons/hopstarter/superhero-avatar/128/Avengers-Hulk-icon.png';//this.props.value.fields.assignee.avatarUrls["48x48"]
        const jiraTicketAssigneeName = 'Leonard Riley';//this.props.value.fields.assignee.name
        const jiraTicketStatusName = 'Open';//this.value.fields.status.name
        const jiraTicketTagStatusKey = 'Open';//this.value.fields.status.statusCategory.key
        const jiraTicketDescription = 'As a user i want to see a preview of message notifications that come in while viewing another screen or channel. current solution: in-app notifications display as banner asdasdas';//this.props.value.fields.description
        const jiraTicketVersions = 'Mobile v2.0';//this.props.value.fields.fixVersions
        const jiraTicketLabels = ['UX Needed', 'Beta'];//this.props.value.fields.labels


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
                                src={jiraStatusIconURI}
                            />
                        </a>
                    </div>
                </div>
                <div className={'ticket-popover-body'}>
                    <div className={'ticket-popover-title'}>
                        <a href={jiraTicketURI}>
                            <h5>{this.truncateString(jiraTicketTitle, 80)}</h5>
                        </a>
                        {this.tagTicketStatus(jiraTicketStatusName, jiraTicketTagStatusKey)}
                    </div>
                    <div
                        className={'ticket-popover-description'}
                        dangerouslySetInnerHTML={{__html: jiraTicketDescription}}
                    />
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
                        is assigned
                    </span>
                </div>
            </div>
        );
    }
}

