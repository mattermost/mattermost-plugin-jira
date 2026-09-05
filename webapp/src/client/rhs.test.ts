// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {RHSIssuesResponse, RHSTab} from 'types/rhs';

import {
    RHSFetchError,
    getRHSIssues,
    getRHSStatuses,
} from './rhs';

const fetchMock = global.fetch as jest.Mock;

const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned to me'};

const emptyIssues: RHSIssuesResponse = {
    issues: [],
    tabs: [assignedTab],
    nextPageToken: '',
    isLast: true,
};

describe('client/rhs', () => {
    beforeEach(() => {
        fetchMock.mockReset();
    });

    afterEach(() => {
        fetchMock.mockReset();
    });

    test('getRHSIssues returns JSON on a 2xx response', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve(emptyIssues),
        }));

        const result = await getRHSIssues('/plugins/jira', {
            instanceID: 'https://x.atlassian.net',
            tab: assignedTab,
            sort: 'updated',
        });
        expect(result).toEqual(emptyIssues);
    });

    test('getRHSIssues surfaces a typed error code from a JSON error body', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 401,
            text: () => Promise.resolve(JSON.stringify({
                error: 'not_connected',
                message: 'Jira account is not connected',
            })),
        }));

        const params = {
            instanceID: 'https://x.atlassian.net',
            tab: assignedTab,
            sort: 'updated' as const,
        };
        await expect(getRHSIssues('/plugins/jira', params)).rejects.toMatchObject({
            name: 'RHSFetchError',
            errorCode: 'not_connected',
            message: 'Jira account is not connected',
            statusCode: 401,
        });
        await expect(getRHSIssues('/plugins/jira', params)).rejects.toBeInstanceOf(RHSFetchError);
    });

    test('getRHSIssues falls back to text when the body is not JSON', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 502,
            text: () => Promise.resolve('<html>bad gateway</html>'),
        }));

        try {
            await getRHSIssues('/plugins/jira', {
                instanceID: 'https://x.atlassian.net',
                tab: assignedTab,
                sort: 'updated',
            });
            throw new Error('expected getRHSIssues to reject');
        } catch (error) {
            expect(error).toBeInstanceOf(RHSFetchError);
            expect(error).not.toBeInstanceOf(SyntaxError);
            expect((error as RHSFetchError).errorCode).toBe('internal_error');
            expect((error as RHSFetchError).message).toBe('<html>bad gateway</html>');
        }
    });

    test('getRHSIssues falls back to text for checkAuth plain 401', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 401,
            text: () => Promise.resolve('Not authorized\n'),
        }));

        try {
            await getRHSIssues('/plugins/jira', {
                instanceID: 'https://x.atlassian.net',
                tab: assignedTab,
                sort: 'updated',
            });
            throw new Error('expected getRHSIssues to reject');
        } catch (error) {
            expect(error).toBeInstanceOf(RHSFetchError);
            expect((error as RHSFetchError).errorCode).toBe('internal_error');
            expect((error as RHSFetchError).message).toContain('Not authorized');
        }
    });

    test('getRHSIssues calls /plugins path with a single tab identity query param', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve(emptyIssues),
        }));

        await getRHSIssues('/plugins/jira', {
            instanceID: 'https://x.atlassian.net',
            tab: {kind: 'category', name: 'In Progress', key: 'indeterminate'},
            sort: 'created',
        });

        expect(fetchMock).toHaveBeenCalledWith(expect.any(String), expect.anything());
        const url = fetchMock.mock.calls[0][0] as string;
        expect(url).toContain('/plugins/jira/api/v2/rhs/issues');
        expect(url).toContain('instance_id=');
        expect(url).toContain('tab=' + encodeURIComponent('category:indeterminate'));
        expect(url).toContain('sort=created');
        expect(url).not.toContain('tab_kind=');
        expect(url).not.toContain('tab_key=');
        expect(url).not.toContain('tab_id=');
        expect(url).not.toContain('next_page_token');
        expect(url).not.toContain('/api/v2/api/v2');
    });

    test('getRHSIssues sends tab=assigned without tab_key or tab_id', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve(emptyIssues),
        }));

        await getRHSIssues('/plugins/jira', {
            instanceID: 'https://x.atlassian.net',
            tab: {kind: 'assigned', name: 'Assigned to me'},
            sort: 'updated',
        });

        const url = fetchMock.mock.calls[0][0] as string;
        expect(url).toContain('tab=assigned');
        expect(url).not.toContain('tab_kind=');
        expect(url).not.toContain('tab_key=');
        expect(url).not.toContain('tab_id=');
    });

    test('getRHSStatuses calls /api/v2/rhs/statuses', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve({statuses: [], categories: []}),
        }));

        await getRHSStatuses('/plugins/jira', 'https://x.atlassian.net');

        expect(fetchMock).toHaveBeenCalledWith(expect.any(String), expect.anything());
        const url = fetchMock.mock.calls[0][0] as string;
        expect(url).toContain('/plugins/jira/api/v2/rhs/statuses');
        expect(url).toContain('instance_id=');
    });
});
