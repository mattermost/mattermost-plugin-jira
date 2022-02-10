import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {id as pluginId} from 'manifest';

import TicketPopover from './jira_ticket_tooltip';

import { isUserConnected ,getUserConnectedInstances ,getInstalledInstances, getIssue ,  getDefaultUserInstanceID} from 'selectors';

import { 
    getIssueByKey,
    getConnected,
} from 'actions'

const mapStateToProps = (state) => {
    return {
        connected: isUserConnected(state),
        isloaded : getIssue(state).isloaded,
        assigneeName:getIssue(state).assigneeName,
        assigneeAvatar:getIssue(state).assigneeAvatar,
        labels:getIssue(state).labels,
        description:getIssue(state).description,
        summary:getIssue(state).summary,
        ticketId:getIssue(state).ticketId,
        jiraIcon:getIssue(state).jiraIcon,
        versions:getIssue(state).versions,
        statusKey:getIssue(state).statusKey,
        issueIcon:getIssue(state).issueIcon,
        defaultUserInstanceID:getDefaultUserInstanceID(state),
        installedInstances:getInstalledInstances(state),
        connectedInstances:getUserConnectedInstances(state)
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    getIssueByKey,
    getConnected
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(TicketPopover);