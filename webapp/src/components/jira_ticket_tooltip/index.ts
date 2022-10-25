import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';

import {isUserConnected, getIssue, getUserConnectedInstances, getDefaultUserInstanceID} from 'selectors';
import {getIssueByKey} from 'actions';

import TicketPopover from './jira_ticket_tooltip';
import {GlobalState} from 'mattermost-redux/types/store';

const mapStateToProps = (state: GlobalState) => {
    return {
        connected: isUserConnected(state),
        ticketDetails: getIssue(state).ticketDetails,
        isLoaded: getIssue(state).isLoaded,
        defaultUserInstanceID: getDefaultUserInstanceID(state),
        connectedInstances: getUserConnectedInstances(state),
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators({
    getIssueByKey,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(TicketPopover);