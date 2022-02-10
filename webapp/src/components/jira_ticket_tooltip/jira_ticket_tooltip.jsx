import React, {Fragment, PureComponent} from 'react';
import './ticketStyle.scss';

import PropTypes from 'prop-types';

export default class TicketPopover extends PureComponent {
    static PropTypes = {
        href: PropTypes.string.isRequired,
        connected: PropTypes.bool.isRequired,
    }

    constructor(props) {
        super(props);
    }

    truncateString(str, num) {
        if (num > str.length) {
            return str;
        }
        return str.substring(0, num) + '...';
    }

    async init() {
        let ticketId = '';
        if (this.props.href.includes('atlassian.net/browse')) {
            ticketId = this.props.href.split('|')[0].split('/browse/')[1];
        }else if (this.href.includes("atlassian.net/jira/software")) {
            const urlParams = new URLSearchParams(fi);
            ticketId = urlParams.get('selectedIssue');
        }
        var instanceID = ""
        if (this.props.connectedInstances.length === 1) {
            instanceID = this.props.connectedInstances[0].instance_id;
        } else if (this.props.defaultUserInstanceID) {
            instanceID = this.props.defaultUserInstanceID;
        }
        if (ticketId != '') {
            this.props.getIssueByKey(ticketId,instanceID)
        }
    }

    componentDidMount() {
        this.props.getConnected()
        if (this.props.connected && !this.props.isloaded) {
            this.init();
        }
    }

    componentDidUpdate(prevProps, prevState) {
        if (this.props.connected !== prevProps.connected) {
            this.setState({connected: this.props.connected}); 
        }
    }

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
        if (!this.props.isloaded){
            return <p></p>
        }

        const jiraTicketURI = this.props.href
        const jiraAvatar = this.props.jiraIcon
        const jiraIssueIconURI = this.props.issueIcon
        const jiraTicketKey = this.props.ticketId
        const jiraTicketTitle = this.props.summary
        const jiraTicketAssigneeAvatarURI = this.props.assigneeAvatar
        const jiraTicketAssigneeName = this.props.assigneeName
        const jiraTicketStatusName = this.props.statusKey
        const jiraTicketDescription = this.props.description
        const jiraTicketVersions = this.props.versions
        const jiraTicketLabels = this.props.labels 


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
                        {this.tagTicketStatus(jiraTicketStatusName, jiraTicketStatusName)}
                    </div>
                    {/* <div
                        className={'ticket-popover-description'}
                        dangerouslySetInnerHTML={{__html: jiraTicketDescription}}
                    /> */}
                    <div
                        className={'ticket-popover-description'} >
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
                        is assigned
                    </span>
                </div>
            </div>
        );
    }
}

