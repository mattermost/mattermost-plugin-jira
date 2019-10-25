// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {doFetchWithResponse, buildQueryString} from 'client';
import {getPluginServerRoute} from 'selectors';
import {ReactSelectOption} from 'types/model';

import JiraEpicSelector from './jira_epic_selector';

const mapStateToProps = (state) => {
    return {
        fetchEpicsWithParams: (params: object): Promise<{data: ReactSelectOption[]}> => {
            const url = getPluginServerRoute(state) + '/api/v2/get-search-epics';
            return doFetchWithResponse(`${url}${buildQueryString(params)}`);
        },
    };
};

export default connect(mapStateToProps)(JiraEpicSelector);
