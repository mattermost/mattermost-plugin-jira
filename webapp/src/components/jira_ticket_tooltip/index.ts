import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {isUserConnected, getUserConnectedInstances, getDefaultUserInstanceID} from 'selectors';

import {getIssueByKey, getConnected} from 'actions';

import TicketPopover from './jira_ticket_tooltip';

const mapStateToProps = (state) => {
    return {
        connected: isUserConnected(state),
        defaultUserInstanceID: getDefaultUserInstanceID(state),
        connectedInstances: getUserConnectedInstances(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    getIssueByKey,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(TicketPopover);