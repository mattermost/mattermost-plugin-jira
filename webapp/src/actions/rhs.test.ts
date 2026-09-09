// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {applyMiddleware, createStore} from 'redux';
import thunk from 'redux-thunk';

import {Instance, InstanceType} from 'types/model';
import {RHSIssuesResponse, RHSTab} from 'types/rhs';
import {pluginStateKey} from 'types/store';
import {saveRHSViewState} from 'utils/rhs_view_state';

import ActionTypes from '../action_types';
import {RHSFetchError} from '../client/rhs';
import reducer from '../reducers';

import {
    fetchRHSIssues,
    loadMoreRHSIssues,
    refreshRHSIssues,
    resolveAndFetchRHSIssues,
    restoreRHSViewState,
} from './rhs';

const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned to me'};
const inProgressTab: RHSTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};
const cloudOAuth = {instance_id: 'https://oauth.example.atlassian.net', type: InstanceType.CLOUD_OAUTH};

const page1: RHSIssuesResponse = {
    issues: [{
        key: 'TES-1',
        summary: 'One',
        browseUrl: 'https://example.atlassian.net/browse/TES-1',
        status: {name: 'In Progress', categoryKey: 'indeterminate'},
        issueType: 'Task',
        project: 'TES',
        updated: '2026-01-02T00:00:00Z',
    }],
    tabs: [assignedTab],
    nextPageToken: 'tok-page-2',
    isLast: false,
};

function makeRHSStore() {
    const pluginState = reducer({} as any, {type: '@@INIT'} as any);
    const initialState = {
        entities: {
            general: {
                config: {
                    SiteURL: 'http://localhost:8065',
                },
            },
            users: {
                currentUserId: 'user-1',
            },
        },
        [pluginStateKey]: pluginState,
    };
    const root = (state = initialState, action: any) => {
        return {
            entities: state.entities,
            [pluginStateKey]: reducer(state[pluginStateKey], action),
        };
    };
    return createStore(root, applyMiddleware(thunk));
}

function mockFetchOk(body: RHSIssuesResponse) {
    const fetchMock = global.fetch as jest.Mock;
    fetchMock.mockImplementation(() => Promise.resolve({
        ok: true,
        json: () => Promise.resolve(body),
    }));
    return fetchMock;
}

function rhsFrom(store: ReturnType<typeof makeRHSStore>) {
    return store.getState()[pluginStateKey].rhs;
}

describe('rhs actions', () => {
    beforeEach(() => {
        (global.fetch as jest.Mock).mockReset();
        localStorage.clear();
    });

    afterEach(() => {
        (global.fetch as jest.Mock).mockReset();
        localStorage.clear();
    });

    test('a slower Assigned fetch does not overwrite a later In Progress fetch', async () => {
        const fetchMock = global.fetch as jest.Mock;
        const resolvers: Array<(value: unknown) => void> = [];
        fetchMock.mockImplementation(() => new Promise((resolve) => {
            resolvers.push(resolve);
        }));

        const store = makeRHSStore();
        const assignedPending = store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);
        const inProgressPending = store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: inProgressTab,
            sort: 'updated',
        }) as any);

        expect(fetchMock).toHaveBeenCalledTimes(2);

        resolvers[1]({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-IP',
                    browseUrl: 'https://example.atlassian.net/browse/TES-IP',
                }],
                tabs: [assignedTab, inProgressTab],
                nextPageToken: '',
                isLast: true,
            }),
        });
        await inProgressPending;

        resolvers[0]({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-ASG',
                    browseUrl: 'https://example.atlassian.net/browse/TES-ASG',
                }],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        });
        await assignedPending;

        const rhs = rhsFrom(store);
        expect(rhs.tab.kind).toBe('category');
        if (rhs.tab.kind === 'category') {
            expect(rhs.tab.key).toBe('indeterminate');
        }
        expect(rhs.issues.map((issue: {key: string}) => issue.key)).toEqual(['TES-IP']);
    });

    test('changing sort resets the list and cursor rather than appending', async () => {
        const fetchMock = mockFetchOk(page1);
        const store = makeRHSStore();

        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        let rhs = rhsFrom(store);
        expect(rhs.issues).toHaveLength(1);
        expect(rhs.nextPageToken).toBe('tok-page-2');
        expect(rhs.isLast).toBe(false);
        expect(rhs.loading).toBe(false);

        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-9',
                    browseUrl: 'https://example.atlassian.net/browse/TES-9',
                }],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        }));

        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'created',
        }) as any);

        rhs = rhsFrom(store);
        expect(rhs.issues).toHaveLength(1);
        expect(rhs.issues[0].key).toBe('TES-9');
        expect(rhs.nextPageToken).toBe('');
    });

    test('loadMoreRHSIssues appends rather than replaces', async () => {
        const fetchMock = mockFetchOk(page1);
        const store = makeRHSStore();

        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-2',
                    browseUrl: 'https://example.atlassian.net/browse/TES-2',
                }],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        }));

        await store.dispatch(loadMoreRHSIssues() as any);

        const rhs = rhsFrom(store);
        expect(rhs.issues.map((issue: {key: string}) => issue.key)).toEqual(['TES-1', 'TES-2']);
        expect(fetchMock.mock.calls[1][0]).toContain('next_page_token=tok-page-2');
        expect(rhs.loading).toBe(false);
    });

    test('loadMoreRHSIssues does not fetch when isLast', async () => {
        const fetchMock = mockFetchOk({
            issues: page1.issues,
            tabs: [assignedTab],
            nextPageToken: '',
            isLast: true,
        });
        const store = makeRHSStore();

        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        await store.dispatch(loadMoreRHSIssues() as any);
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    test('load-more result does not append onto a tab fetched while it was in flight', async () => {
        const fetchMock = mockFetchOk(page1);
        const store = makeRHSStore();

        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        const resolvers: Array<(value: unknown) => void> = [];
        fetchMock.mockImplementation(() => new Promise((resolve) => {
            resolvers.push(resolve);
        }));

        const loadMorePending = store.dispatch(loadMoreRHSIssues() as any);
        const inProgressPending = store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: inProgressTab,
            sort: 'updated',
        }) as any);

        expect(fetchMock).toHaveBeenCalledTimes(3);

        resolvers[1]({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-IP',
                    browseUrl: 'https://example.atlassian.net/browse/TES-IP',
                }],
                tabs: [assignedTab, inProgressTab],
                nextPageToken: '',
                isLast: true,
            }),
        });
        await inProgressPending;

        resolvers[0]({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-2',
                    browseUrl: 'https://example.atlassian.net/browse/TES-2',
                }],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        });
        await loadMorePending;

        const rhs = rhsFrom(store);
        if (rhs.tab.kind === 'category') {
            expect(rhs.tab.key).toBe('indeterminate');
        } else {
            throw new Error('expected category tab');
        }
        expect(rhs.issues.map((issue: {key: string}) => issue.key)).toEqual(['TES-IP']);
    });

    test('refresh while load-more is in flight issues a new page-1 request and ignores the append', async () => {
        const fetchMock = mockFetchOk(page1);
        const store = makeRHSStore();

        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        const resolvers: Array<(value: unknown) => void> = [];
        fetchMock.mockImplementation(() => new Promise((resolve) => {
            resolvers.push(resolve);
        }));

        const loadMorePending = store.dispatch(loadMoreRHSIssues() as any);
        const refreshPending = store.dispatch(refreshRHSIssues() as any);

        expect(fetchMock).toHaveBeenCalledTimes(3);
        expect(String(fetchMock.mock.calls[1][0])).toContain('next_page_token=tok-page-2');
        expect(String(fetchMock.mock.calls[2][0])).not.toContain('next_page_token');

        resolvers[1]({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-9',
                    browseUrl: 'https://example.atlassian.net/browse/TES-9',
                }],
                tabs: [assignedTab],
                nextPageToken: 'tok-refresh',
                isLast: false,
            }),
        });
        await refreshPending;

        resolvers[0]({
            ok: true,
            json: () => Promise.resolve({
                issues: [{
                    ...page1.issues[0],
                    key: 'TES-2',
                    browseUrl: 'https://example.atlassian.net/browse/TES-2',
                }],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        });
        await loadMorePending;

        const rhs = rhsFrom(store);
        expect(rhs.issues.map((issue: {key: string}) => issue.key)).toEqual(['TES-9']);
        expect(rhs.nextPageToken).toBe('tok-refresh');
    });

    test('loading is true and issues are empty during a page-1 fetch', async () => {
        const fetchMock = global.fetch as jest.Mock;
        let resolveFetch: (value: unknown) => void = () => {
            // filled in by the mock
        };
        fetchMock.mockImplementation(() => new Promise((resolve) => {
            resolveFetch = resolve;
        }));

        const store = makeRHSStore();
        const pending = store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        const rhs = rhsFrom(store);
        expect(rhs.loading).toBe(true);
        expect(rhs.issues).toHaveLength(0);
        expect(rhs.error).toBeNull();

        resolveFetch({
            ok: true,
            json: () => Promise.resolve({
                issues: [],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        });
        await pending;
    });

    test('a JSON not_connected error stores the typed code', async () => {
        const fetchMock = global.fetch as jest.Mock;
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 401,
            text: () => Promise.resolve(JSON.stringify({
                error: 'not_connected',
                message: 'Jira account is not connected',
            })),
        }));

        const store = makeRHSStore();
        const result = await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        const rhs = rhsFrom(store);
        expect(rhs.error).toBe('not_connected');
        expect(rhs.loading).toBe(false);
        expect(result.error).toBeInstanceOf(RHSFetchError);
        expect(result.error.errorCode).toBe('not_connected');
    });

    test('fetchRHSIssues persists view state keyed by user id', async () => {
        mockFetchOk({
            issues: [],
            tabs: [assignedTab],
            nextPageToken: '',
            isLast: true,
        });
        const store = makeRHSStore();
        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'created',
        }) as any);

        const stored = JSON.parse(localStorage.getItem('jira:rhs-view:user-1') as string);
        expect(stored.sort).toBe('created');
        expect(stored.instance).toBe('https://example.atlassian.net');
        expect(stored.tab.kind).toBe('assigned');
    });

    test('restoreRHSViewState hydrates instance tab and sort without fetching', () => {
        saveRHSViewState('user-1', {
            instance: 'https://oauth.example.atlassian.net',
            tab: inProgressTab,
            sort: 'created',
        });

        const store = makeRHSStore();
        store.dispatch(restoreRHSViewState());

        const rhs = rhsFrom(store);
        expect(rhs.instanceID).toBe('https://oauth.example.atlassian.net');
        expect(rhs.sort).toBe('created');
        expect(rhs.tab).toEqual(inProgressTab);
        expect(rhs.issues).toEqual([]);
        expect(global.fetch as jest.Mock).not.toHaveBeenCalled();
    });

    test('resolveAndFetchRHSIssues does not fetch when no connected Cloud instance', async () => {
        const fetchMock = global.fetch as jest.Mock;
        const store = makeRHSStore();

        await store.dispatch(resolveAndFetchRHSIssues() as any);

        expect(fetchMock).toHaveBeenCalledTimes(0);
    });

    test('resolveAndFetchRHSIssues does not retry Assigned after invalid_request', async () => {
        const vanishedTab: RHSTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};
        const fetchMock = global.fetch as jest.Mock;
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 400,
            text: () => Promise.resolve(JSON.stringify({
                error: 'invalid_request',
                message: 'unknown tab',
            })),
        }));

        const store = makeRHSStore();
        seedConnectedCloud(store);

        await store.dispatch(resolveAndFetchRHSIssues({tab: vanishedTab}) as any);

        expect(fetchMock).toHaveBeenCalledTimes(1);
        expect(rhsFrom(store).error).toBe('invalid_request');
    });

    test('hydrated view state triggers a fresh issues fetch with the tab identity', async () => {
        const saved = {
            instance: 'https://oauth.example.atlassian.net',
            tab: inProgressTab,
            sort: 'created' as const,
        };
        saveRHSViewState('user-1', saved);

        const stored = JSON.parse(localStorage.getItem('jira:rhs-view:user-1') as string);
        expect(stored.issues).toBeUndefined();
        expect(stored.instance).toBe(saved.instance);

        const store = makeRHSStore();
        store.dispatch({
            type: ActionTypes.RECEIVED_INSTANCE_STATUS,
            data: {instances: [cloudOAuth]},
        });
        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: userinfoWithInstances([cloudOAuth]),
        });

        store.dispatch(restoreRHSViewState());
        expect(rhsFrom(store).instanceID).toBe(saved.instance);
        expect(rhsFrom(store).sort).toBe('created');
        expect(rhsFrom(store).issues).toEqual([]);

        const fetchMock = mockFetchOk(page1);
        await store.dispatch(resolveAndFetchRHSIssues() as any);
        expect(fetchMock).toHaveBeenCalledTimes(1);
        const url = String(fetchMock.mock.calls[0][0]);
        expect(url).toContain('/api/v2/rhs/issues');
        expect(url).toContain('sort=created');
        expect(url).toContain('tab=' + encodeURIComponent('category:indeterminate'));
        expect(url).not.toContain('tab_kind=');
        expect(rhsFrom(store).issues.length).toBeGreaterThan(0);
    });
});

function userinfoWithInstances(instances: Instance[]) {
    return {
        can_connect: true,
        is_connected: instances.length > 0,
        instances,
        user_info: {
            connected_instances: instances,
            default_instance_id: instances[0] ? instances[0].instance_id : '',
        },
    };
}

function seedConnectedCloud(store: ReturnType<typeof makeRHSStore>) {
    const instance = {instance_id: 'https://example.atlassian.net', type: InstanceType.CLOUD};
    store.dispatch({
        type: ActionTypes.RECEIVED_INSTANCE_STATUS,
        data: {instances: [instance]},
    });
    store.dispatch({
        type: ActionTypes.RECEIVED_CONNECTED,
        data: {
            can_connect: true,
            is_connected: true,
            user_info: {
                connected_instances: [instance],
                default_instance_id: instance.instance_id,
            },
        },
    });
}
