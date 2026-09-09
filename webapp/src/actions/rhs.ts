// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Dispatch} from 'redux';

import ActionTypes from '../action_types';
import {RHSFetchError, getRHSIssues} from '../client/rhs';
import {getConnectedCloudInstances, getDefaultUserInstanceID, getPluginServerRoute} from '../selectors';
import {resolveRHSInstanceID} from 'utils/rhs_resolve';
import {loadRHSViewState, saveRHSViewState} from 'utils/rhs_view_state';

import {
    FetchRHSIssuesArgs,
    RHSSort,
    RHSTab,
} from 'types/rhs';
import {GlobalState, pluginStateKey} from 'types/store';

function persistCurrentRHSView(state: GlobalState): void {
    const rhs = state[pluginStateKey].rhs;
    saveRHSViewState(state.entities.users.currentUserId, {
        instance: rhs.instanceID,
        tab: rhs.tab,
        sort: rhs.sort,
    });
}

function dispatchRHSFetchError(dispatch: Dispatch, error: unknown, requestId: number) {
    const rhsError = error instanceof RHSFetchError ?
        error :
        new RHSFetchError('internal_error', error instanceof Error ? error.message : 'internal error', 0);
    dispatch({
        type: ActionTypes.RHS_ISSUES_ERROR,
        data: rhsError.errorCode,
        requestId,
    });
    return {error: rhsError};
}

export const restoreRHSViewState = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const saved = loadRHSViewState(getState().entities.users.currentUserId);
        if (!saved) {
            return {data: null};
        }
        dispatch({
            type: ActionTypes.SET_RHS_VIEW,
            data: {
                instanceID: saved.instance,
                tab: saved.tab,
                sort: saved.sort,
            },
        });
        return {data: saved};
    };
};

export const fetchRHSIssues = (args: FetchRHSIssuesArgs) => {
    return async (dispatch: Dispatch, getState: () => GlobalState) => {
        dispatch({
            type: ActionTypes.SET_RHS_VIEW,
            data: {
                instanceID: args.instanceID,
                tab: args.tab,
                sort: args.sort,
            },
        });
        persistCurrentRHSView(getState());
        dispatch({
            type: ActionTypes.RHS_ISSUES_LOADING,
            data: {reset: true},
        });
        const requestId = getState()[pluginStateKey].rhs.requestId;

        try {
            const data = await getRHSIssues(getPluginServerRoute(getState()), {
                instanceID: args.instanceID,
                tab: args.tab,
                sort: args.sort,
            });
            dispatch({
                type: ActionTypes.RECEIVED_RHS_ISSUES,
                data,
                requestId,
            });
            return {data};
        } catch (error) {
            return dispatchRHSFetchError(dispatch, error, requestId);
        }
    };
};

export const loadMoreRHSIssues = () => {
    return async (dispatch: Dispatch, getState: () => GlobalState) => {
        const rhs = getState()[pluginStateKey].rhs;
        if (rhs.isLast || !rhs.nextPageToken) {
            return {data: rhs.issues};
        }

        dispatch({
            type: ActionTypes.RHS_ISSUES_LOADING,
            data: {reset: false},
        });
        const requestId = getState()[pluginStateKey].rhs.requestId;

        try {
            const data = await getRHSIssues(getPluginServerRoute(getState()), {
                instanceID: rhs.instanceID,
                tab: rhs.tab,
                sort: rhs.sort,
                nextPageToken: rhs.nextPageToken,
            });
            dispatch({
                type: ActionTypes.RECEIVED_RHS_ISSUES_APPEND,
                data,
                requestId,
            });
            return {data};
        } catch (error) {
            return dispatchRHSFetchError(dispatch, error, requestId);
        }
    };
};

export const refreshRHSIssues = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const rhs = getState()[pluginStateKey].rhs;
        return fetchRHSIssues({
            instanceID: rhs.instanceID,
            tab: rhs.tab,
            sort: rhs.sort,
        })(dispatch, getState);
    };
};

export type ResolveAndFetchArgs = {
    instanceID?: string;
    tab?: RHSTab;
    sort?: RHSSort;
};

export const resolveAndFetchRHSIssues = (overrides: ResolveAndFetchArgs = {}) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const state = getState();
        const rhs = state[pluginStateKey].rhs;
        const instanceID = overrides.instanceID || resolveRHSInstanceID(
            rhs.instanceID,
            getConnectedCloudInstances(state),
            getDefaultUserInstanceID(state) || '',
        );
        if (!instanceID) {
            return Promise.resolve({data: null});
        }

        const tab = overrides.tab || rhs.tab;
        const sort = overrides.sort || rhs.sort;

        return fetchRHSIssues({instanceID, tab, sort})(dispatch, getState);
    };
};
