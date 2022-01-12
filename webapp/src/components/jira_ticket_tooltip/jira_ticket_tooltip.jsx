import React, {Fragment, PureComponent} from "react";
import JiraAvatar from './assets/jira_avatar.png';
import PropTypes from 'prop-types';

export default class TicketPopover extends PureComponent {
    static propTypes = {
        href: PropTypes.string,
        connected: PropTypes.any,
        value: PropTypes.any,
    }


    truncateString(str, num) {
        if (num > str.length) {
            return str;
        } else {
            str = str.substring(0, num);
            return str + "...";
        }
    }

    async init(){
        let ticketId = ""
        if (this.href.includes('attlasian.net/browse')){
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
    fixVersionLabel(fixVersion) {
        if (fixVersion) {
            return <div className="fix-version-label" style={{
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
            </span></div>;
        }
    }

    tagTicketStatus(ticketStatus, ticketStatusKey) {
        const defaultStyle = {
            fontFamily: 'open sans',
            fontStyle: 'normal',
            fontWeight: '600',
            fontSize: '12px',
            marginTop: '4px',
            padding: '4px 8px 0px 8px',
            align: 'center',
            height: 20,
            marginBottom: '8px',
            borderRadius: '4px',
        }
        if (ticketStatusKey === "indeterminate") {
            console.log(ticketStatusKey)
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
        return (
            <div className={'ticket-popover'}>
                <div className={'ticket-popover-header'}>
                    <div className={'ticket-popover-header-container'}>
                        <a href={this.props.value.self} title={'goto ticket'}>
                            <img src={JiraAvatar} width={14} height={14}
                                 alt={'jira-avatar'}
                                 className={'ticket-popover-header-avatar'}/></a>
                        <a href={this.props.value.self} className={'ticket-popover-keyword'}>
                            <span style={{fontSize: 12}}>{this.props.value.key}</span>
                            <img alt={'jira-issue-icon'} width="14" height="14" src={this.props.value.fields.status.iconUrl}/>
                        </a>
                    </div>
                </div>
                <div className={'ticket-popover-body'}>
                    <div className={'ticket-popover-title'}>
                        <a href={this.props.value.self}>
                            <h5>{this.truncateString(this.props.value.names, 80)}</h5>
                        </a>
                        {this.tagTicketStatus(this.value.fields.status.name, this.value.fields.status.statusCategory.key)}
                    </div>
                    <div className={'ticket-popover-description'}
                         dangerouslySetInnerHTML={{__html: this.props.value.fields.description}}/>
                    <div className={'ticket-popover-labels'}>
                        {this.fixVersionLabel(this.props.value.fields.fixVersions)}
                        {this.labelList(this.props.value.fields.labels)}
                    </div>
                </div>
                <div className={'ticket-popover-footer'}>
                    <img className={'ticket-popover-footer-assigner-profile'} src={this.props.value.fields.assignee.avatarUrls["48x48"]}
                         alt={'jira assigner profile'}/>
                    <span className={'ticket-popover-footer-assigner-name'}>
                            {this.props.value.fields.assignee.name}
                        </span>
                    <span className={'ticket-popover-footer-assigner-is-assigned'}>is assigned</span>
                </div>
            </div>
        )
    }
}

