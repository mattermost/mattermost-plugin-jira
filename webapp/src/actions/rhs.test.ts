// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {applyMiddleware, createStore} from 'redux';
import thunk from 'redux-thunk';

import {RHSIssuesResponse, RHSTab} from 'types/model';
import {pluginStateKey} from 'types/store';

import {RHSFetchError} from '../client';
import reducer from '../reducers';

import {
    fetchRHSIssues,
    fetchRHSStatuses,
    loadMoreRHSIssues,
    resetRHSIssuesInFlight,
    setRHSSort,
} from './rhs';

const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned'};

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
});
