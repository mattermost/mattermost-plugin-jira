// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {applyMiddleware, createStore} from 'redux';
import thunk from 'redux-thunk';

import {
    Instance,
    InstanceType,
    RHSIssuesResponse,
    RHSTab,
} from 'types/model';
import {pluginStateKey} from 'types/store';
import {isRHSPopoutPathname} from 'utils/rhs_popout';
import {saveRHSViewState} from 'utils/rhs_view_state';

import ActionTypes from '../action_types';
import {RHSFetchError} from '../client';
import reducer from '../reducers';

import {
    fetchRHSIssues,
    fetchRHSStatuses,
    loadMoreRHSIssues,
    refreshRHSIssues,
    resetRHSIssuesInFlight,
    resolveAndFetchRHSIssues,
    restoreRHSViewState,
    setRHSSort,
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
        priority: 'Medium',
        issueType: 'Task',
        project: 'TES',
        assignee: 'alice',
        reporter: 'bob',
        created: '2026-01-01T00:00:00Z',
        updated: '2026-01-02T00:00:00Z',
        dueDate: '',
        labels: [],
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

function pluginFrom(store: ReturnType<typeof makeRHSStore>) {
    return store.getState()[pluginStateKey];
}

describe('rhs actions', () => {
    beforeEach(() => {
        resetRHSIssuesInFlight();
        (global.fetch as jest.Mock).mockReset();
        localStorage.clear();
    });

    afterEach(() => {
        resetRHSIssuesInFlight();
        (global.fetch as jest.Mock).mockReset();
        localStorage.clear();
    });

    test('a second fetchRHSIssues for the same instance tab sort while in flight issues no second network call', async () => {
        const fetchMock = global.fetch as jest.Mock;
        let resolveFetch: (value: unknown) => void = () => {
            // filled in by the mock
        };
        fetchMock.mockImplementation(() => new Promise((resolve) => {
            resolveFetch = resolve;
        }));

        const store = makeRHSStore();
        const args = {instanceID: 'https://example.atlassian.net', tab: assignedTab, sort: 'updated' as const};

        const p1 = store.dispatch(fetchRHSIssues(args) as any);
        const p2 = store.dispatch(fetchRHSIssues(args) as any);

        expect(fetchMock).toHaveBeenCalledTimes(1);

        resolveFetch({
            ok: true,
            json: () => Promise.resolve({
                issues: [],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        });

        await p1;
        await p2;
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    test('fetchRHSIssues for a different tab while in flight does call fetch again', async () => {
        const fetchMock = global.fetch as jest.Mock;
        const resolvers: Array<(value: unknown) => void> = [];
        fetchMock.mockImplementation(() => new Promise((resolve) => {
            resolvers.push(resolve);
        }));

        const store = makeRHSStore();
        const p1 = store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);
        const p2 = store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: {kind: 'category', name: 'In Progress', key: 'indeterminate'},
            sort: 'updated',
        }) as any);

        expect(fetchMock).toHaveBeenCalledTimes(2);

        const empty = {
            ok: true,
            json: () => Promise.resolve({
                issues: [],
                tabs: [assignedTab],
                nextPageToken: '',
                isLast: true,
            }),
        };
        resolvers.forEach((resolve) => resolve(empty));
        await p1;
        await p2;
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

        const plugin = pluginFrom(store);
        expect(plugin.rhsTab.kind).toBe('category');
        expect(plugin.rhsTab.key).toBe('indeterminate');
        expect(plugin.rhsIssues.map((issue: {key: string}) => issue.key)).toEqual(['TES-IP']);
    });

    test('fetchRHSIssues after the in-flight request settles does call fetch again', async () => {
        const fetchMock = mockFetchOk({
            issues: [],
            tabs: [assignedTab],
            nextPageToken: '',
            isLast: true,
        });
        const store = makeRHSStore();
        const args = {instanceID: 'https://example.atlassian.net', tab: assignedTab, sort: 'updated' as const};

        await store.dispatch(fetchRHSIssues(args) as any);
        await store.dispatch(fetchRHSIssues(args) as any);

        expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    test('changing sort resets the list and cursor rather than appending', async () => {
        const fetchMock = mockFetchOk(page1);
        const store = makeRHSStore();

        await store.dispatch(fetchRHSIssues({
            instanceID: 'https://example.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        }) as any);

        let plugin = pluginFrom(store);
        expect(plugin.rhsIssues).toHaveLength(1);
        expect(plugin.rhsNextPageToken).toBe('tok-page-2');
        expect(plugin.rhsIsLast).toBe(false);
        expect(plugin.rhsLoading).toBe(false);

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

        plugin = pluginFrom(store);
        expect(plugin.rhsIssues).toHaveLength(1);
        expect(plugin.rhsIssues[0].key).toBe('TES-9');
        expect(plugin.rhsNextPageToken).toBe('');
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

        const plugin = pluginFrom(store);
        expect(plugin.rhsIssues.map((issue: {key: string}) => issue.key)).toEqual(['TES-1', 'TES-2']);
        expect(fetchMock.mock.calls[1][0]).toContain('next_page_token=tok-page-2');
        expect(plugin.rhsLoading).toBe(false);
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

        const plugin = pluginFrom(store);
        expect(plugin.rhsTab.key).toBe('indeterminate');
        expect(plugin.rhsIssues.map((issue: {key: string}) => issue.key)).toEqual(['TES-IP']);
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

        const plugin = pluginFrom(store);
        expect(plugin.rhsIssues.map((issue: {key: string}) => issue.key)).toEqual(['TES-9']);
        expect(plugin.rhsNextPageToken).toBe('tok-refresh');
    });

    test('rhsLoading is true and issues are empty during a page-1 fetch', async () => {
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

        const plugin = pluginFrom(store);
        expect(plugin.rhsLoading).toBe(true);
        expect(plugin.rhsIssues).toHaveLength(0);
        expect(plugin.rhsError).toBeNull();

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

        const plugin = pluginFrom(store);
        expect(plugin.rhsError).toBe('not_connected');
        expect(plugin.rhsLoading).toBe(false);
        expect(result.error).toBeInstanceOf(RHSFetchError);
        expect(result.error.errorCode).toBe('not_connected');
    });

    test('fetchRHSStatuses returns data without writing issue slices', async () => {
        const fetchMock = global.fetch as jest.Mock;
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve({statuses: [], categories: []}),
        }));

        const store = makeRHSStore();
        const result = await store.dispatch(fetchRHSStatuses('https://example.atlassian.net') as any);

        const plugin = pluginFrom(store);
        expect(plugin.rhsIssues).toEqual([]);
        expect(plugin.rhsLoading).toBe(false);
        expect(result).toEqual({data: {statuses: [], categories: []}});
    });

    test('setRHSSort persists view state keyed by user id', () => {
        const store = makeRHSStore();
        store.dispatch(setRHSSort('created') as any);

        const stored = JSON.parse(localStorage.getItem('jira:rhs-view:user-1') as string);
        expect(stored.sort).toBe('created');
    });

    test('resolveAndFetchRHSIssues does not fetch when no connected Cloud instance', async () => {
        const fetchMock = global.fetch as jest.Mock;
        const store = makeRHSStore();

        await store.dispatch(resolveAndFetchRHSIssues() as any);

        expect(fetchMock).toHaveBeenCalledTimes(0);
    });

    test('resolveAndFetchRHSIssues retries Assigned after invalid_request on a vanished tab', async () => {
        const vanishedTab: RHSTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};
        const fetchMock = global.fetch as jest.Mock;
        fetchMock
            .mockImplementationOnce(() => Promise.resolve({
                ok: false,
                status: 400,
                text: () => Promise.resolve(JSON.stringify({
                    error: 'invalid_request',
                    message: 'unknown tab',
                })),
            }))
            .mockImplementationOnce(() => Promise.resolve({
                ok: true,
                json: () => Promise.resolve({
                    issues: [],
                    tabs: [assignedTab],
                    nextPageToken: '',
                    isLast: true,
                }),
            }));

        const store = makeRHSStore();
        seedConnectedCloud(store);

        await store.dispatch(resolveAndFetchRHSIssues({tab: vanishedTab}) as any);

        expect(fetchMock).toHaveBeenCalledTimes(2);
        const plugin = pluginFrom(store);
        expect(plugin.rhsTab.kind).toBe('assigned');
        expect(plugin.rhsError).toBeNull();
    });

    test('resolveAndFetchRHSIssues does not retry Assigned when the first error is invalid_request on Assigned', async () => {
        const fetchMock = global.fetch as jest.Mock;
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 400,
            text: () => Promise.resolve(JSON.stringify({
                error: 'invalid_request',
                message: 'bad assigned request',
            })),
        }));

        const store = makeRHSStore();
        seedConnectedCloud(store);

        await store.dispatch(resolveAndFetchRHSIssues() as any);

        expect(fetchMock).toHaveBeenCalledTimes(1);
        expect(pluginFrom(store).rhsError).toBe('invalid_request');
    });

    test('a simulated /_popout/ pathname rehydrates the persisted view state and triggers a fresh issues fetch', async () => {
        expect(isRHSPopoutPathname('/_popout/rhs/team/plugin/jira')).toBe(true);

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
        expect(pluginFrom(store).rhsInstanceID).toBe(saved.instance);
        expect(pluginFrom(store).rhsSort).toBe('created');
        expect(pluginFrom(store).rhsIssues).toEqual([]);

        const fetchMock = mockFetchOk(page1);
        await store.dispatch(resolveAndFetchRHSIssues() as any);
        expect(fetchMock).toHaveBeenCalledTimes(1);
        const url = String(fetchMock.mock.calls[0][0]);
        expect(url).toContain('/api/v2/rhs/issues');
        expect(url).toContain('sort=created');
        expect(url).toContain('tab_kind=category');
        expect(url).toContain('tab_key=indeterminate');
        expect(pluginFrom(store).rhsIssues.length).toBeGreaterThan(0);
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
