import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';
import {GlobalState} from 'mattermost-redux/types/store';

import {isUserConnected, getIssue, getUserConnectedInstances, getDefaultUserInstanceID} from 'selectors';
import {fetchIssueByKey} from 'actions';

import TicketPopover from './jira_ticket_tooltip';

const mapStateToProps = (state: GlobalState) => {
    return {
        connected: isUserConnected(state),
        ticketDetails: getIssue(state).ticketDetails,
        connectedInstances: getUserConnectedInstances(state),
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators({
    fetchIssueByKey,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(TicketPopover);
