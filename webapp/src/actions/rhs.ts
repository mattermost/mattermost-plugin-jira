// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Dispatch} from 'redux';

import ActionTypes from '../action_types';
import {RHSFetchError, getRHSIssues, getRHSStatuses} from '../client';
import {getConnectedCloudInstances, getDefaultUserInstanceID, getPluginServerRoute} from '../selectors';
import {resolveRHSInstanceID} from 'utils/rhs_resolve';
import {loadRHSViewState, saveRHSViewState} from 'utils/rhs_view_state';

import {
    FetchRHSIssuesArgs,
    RHSErrorCode,
    RHSIssuesResponse,
    RHSSort,
    RHSTab,
    RHS_DEFAULT_TAB,
} from 'types/model';
import {GlobalState, pluginStateKey} from 'types/store';

export function rhsIssuesFlightKey(instanceID: string, tab: RHSTab, sort: RHSSort, mode: 'reset' | 'append' = 'reset'): string {
    return JSON.stringify({
        instance: instanceID,
        kind: tab.kind,
        key: tab.key || '',
        id: tab.id || '',
        sort,
        mode,
    });
}

const rhsIssuesInFlight: Map<string, Promise<{data: RHSIssuesResponse} | {error: RHSFetchError}>> = new Map();
let rhsIssuesGeneration = 0;

export function resetRHSIssuesInFlight(): void {
    rhsIssuesInFlight.clear();
    rhsIssuesGeneration = 0;
}

function isCurrentRHSGeneration(gen: number): boolean {
    return gen === rhsIssuesGeneration;
}

function withRHSIssuesInFlight(
    key: string,
    work: () => Promise<{data: RHSIssuesResponse} | {error: RHSFetchError}>,
): Promise<{data: RHSIssuesResponse} | {error: RHSFetchError}> {
    const existing = rhsIssuesInFlight.get(key);
    if (existing) {
        return existing;
    }

    const promise = (async () => {
        try {
            return await work();
        } finally {
            rhsIssuesInFlight.delete(key);
        }
    })();

    rhsIssuesInFlight.set(key, promise);
    return promise;
}

function persistCurrentRHSView(state: GlobalState): void {
    const plugin = state[pluginStateKey];
    saveRHSViewState(state.entities.users.currentUserId, {
        instance: plugin.rhsInstanceID,
        tab: plugin.rhsTab,
        sort: plugin.rhsSort,
    });
}

function toRHSFetchError(error: unknown): RHSFetchError {
    if (error instanceof RHSFetchError) {
        return error;
    }
    const message = error instanceof Error ? error.message : 'internal error';
    const errorCode: RHSErrorCode = 'internal_error';
    return new RHSFetchError(errorCode, message, 0);
}

export const setRHSInstanceID = (instanceID: string) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        dispatch({
            type: ActionTypes.SET_RHS_INSTANCE_ID,
            data: instanceID,
        });
        persistCurrentRHSView(getState());
        return {data: instanceID};
    };
};

export const setRHSTab = (tab: RHSTab) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        dispatch({
            type: ActionTypes.SET_RHS_TAB,
            data: tab,
        });
        persistCurrentRHSView(getState());
        return {data: tab};
    };
};

export const setRHSSort = (sort: RHSSort) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        dispatch({
            type: ActionTypes.SET_RHS_SORT,
            data: sort,
        });
        persistCurrentRHSView(getState());
        return {data: sort};
    };
};

export const restoreRHSViewState = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const saved = loadRHSViewState(getState().entities.users.currentUserId);
        if (!saved) {
            return {data: null};
        }
        dispatch({
            type: ActionTypes.HYDRATE_RHS_VIEW_STATE,
            data: saved,
        });
        return {data: saved};
    };
};

export const fetchRHSIssues = (args: FetchRHSIssuesArgs) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const key = rhsIssuesFlightKey(args.instanceID, args.tab, args.sort, 'reset');

        return withRHSIssuesInFlight(key, async () => {
            const gen = ++rhsIssuesGeneration;
            if (isCurrentRHSGeneration(gen)) {
                dispatch({
                    type: ActionTypes.SET_RHS_INSTANCE_ID,
                    data: args.instanceID,
                });
                dispatch({
                    type: ActionTypes.SET_RHS_TAB,
                    data: args.tab,
                });
                dispatch({
                    type: ActionTypes.SET_RHS_SORT,
                    data: args.sort,
                });
                persistCurrentRHSView(getState());
                dispatch({
                    type: ActionTypes.RHS_ISSUES_LOADING,
                    data: {reset: true},
                });
            }

            try {
                const data = await getRHSIssues(getPluginServerRoute(getState()), {
                    instanceID: args.instanceID,
                    tab: args.tab,
                    sort: args.sort,
                });
                if (!isCurrentRHSGeneration(gen)) {
                    return {data};
                }
                dispatch({
                    type: ActionTypes.RECEIVED_RHS_ISSUES,
                    data,
                });
                return {data};
            } catch (error) {
                const rhsError = toRHSFetchError(error);
                if (!isCurrentRHSGeneration(gen)) {
                    return {error: rhsError};
                }
                dispatch({
                    type: ActionTypes.RHS_ISSUES_ERROR,
                    data: rhsError.errorCode,
                });
                return {error: rhsError};
            }
        });
    };
};

export const loadMoreRHSIssues = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const plugin = getState()[pluginStateKey];
        if (plugin.rhsIsLast || !plugin.rhsNextPageToken) {
            return Promise.resolve({data: plugin.rhsIssues});
        }

        const instanceID = plugin.rhsInstanceID;
        const tab = plugin.rhsTab;
        const sort = plugin.rhsSort;
        const nextPageToken = plugin.rhsNextPageToken;
        const key = rhsIssuesFlightKey(instanceID, tab, sort, 'append');
        const gen = rhsIssuesGeneration;

        return withRHSIssuesInFlight(key, async () => {
            if (isCurrentRHSGeneration(gen)) {
                dispatch({
                    type: ActionTypes.RHS_ISSUES_LOADING,
                    data: {reset: false},
                });
            }

            try {
                const data = await getRHSIssues(getPluginServerRoute(getState()), {
                    instanceID,
                    tab,
                    sort,
                    nextPageToken,
                });
                if (!isCurrentRHSGeneration(gen)) {
                    return {data};
                }
                dispatch({
                    type: ActionTypes.RECEIVED_RHS_ISSUES_APPEND,
                    data,
                });
                return {data};
            } catch (error) {
                const rhsError = toRHSFetchError(error);
                if (!isCurrentRHSGeneration(gen)) {
                    return {error: rhsError};
                }
                dispatch({
                    type: ActionTypes.RHS_ISSUES_ERROR,
                    data: rhsError.errorCode,
                });
                return {error: rhsError};
            }
        });
    };
};

export const refreshRHSIssues = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const plugin = getState()[pluginStateKey];
        return fetchRHSIssues({
            instanceID: plugin.rhsInstanceID,
            tab: plugin.rhsTab,
            sort: plugin.rhsSort,
        })(dispatch, getState);
    };
};

export const fetchRHSStatuses = (instanceID: string) => {
    return async (dispatch: Dispatch, getState: () => GlobalState) => {
        try {
            const data = await getRHSStatuses(getPluginServerRoute(getState()), instanceID);
            return {data};
        } catch (error) {
            return {error: toRHSFetchError(error)};
        }
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
        const plugin = state[pluginStateKey];
        const instanceID = overrides.instanceID || resolveRHSInstanceID(
            plugin.rhsInstanceID,
            getConnectedCloudInstances(state),
            getDefaultUserInstanceID(state) || '',
        );
        if (!instanceID) {
            return Promise.resolve({data: null});
        }

        const tab = overrides.tab || plugin.rhsTab;
        const sort = overrides.sort || plugin.rhsSort;

        return fetchRHSIssues({instanceID, tab, sort})(dispatch, getState).then((result) => {
            if (
                result &&
                'error' in result &&
                result.error &&
                result.error.errorCode === 'invalid_request' &&
                tab.kind !== 'assigned'
            ) {
                return fetchRHSIssues({
                    instanceID,
                    tab: RHS_DEFAULT_TAB,
                    sort,
                })(dispatch, getState);
            }
            return result;
        });
    };
};
