import {connect} from 'react-redux';
import {Dispatch, bindActionCreators} from 'redux';
import {GlobalState} from 'mattermost-redux/types/store';

import {getUserConnectedInstances, isUserConnected} from 'selectors';
import {fetchIssueByKey} from 'actions';

import TicketPopover from './jira_ticket_tooltip';

const mapStateToProps = (state: GlobalState) => {
    return {
        connected: isUserConnected(state),
        connectedInstances: getUserConnectedInstances(state),
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => bindActionCreators({
    fetchIssueByKey,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(TicketPopover);
