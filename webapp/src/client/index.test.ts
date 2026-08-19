// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ClientError} from '@mattermost/client';

import {RHSIssuesResponse, RHSTab} from 'types/model';

import {
    RHSFetchError,
    doFetchJSON,
    doFetchWithResponse,
    getRHSIssues,
    getRHSStatuses,
} from './index';

const fetchMock = global.fetch as jest.Mock;

const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned'};

const emptyIssues: RHSIssuesResponse = {
    issues: [],
    tabs: [assignedTab],
    nextPageToken: '',
    isLast: true,
};

describe('client JSON helper', () => {
    beforeEach(() => {
        fetchMock.mockReset();
    });

    afterEach(() => {
        fetchMock.mockReset();
    });

    test('doFetchJSON returns JSON on a 2xx response', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve({hello: 'world'}),
        }));

        const result = await doFetchJSON('/plugins/jira/api/v2/rhs/issues', {method: 'get'});
        expect(result).toEqual({hello: 'world'});
    });

    test('doFetchJSON surfaces a typed error code from a JSON error body', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 401,
            text: () => Promise.resolve(JSON.stringify({
                error: 'not_connected',
                message: 'Jira account is not connected',
            })),
        }));

        await expect(doFetchJSON('/plugins/jira/api/v2/rhs/issues', {method: 'get'})).rejects.toMatchObject({
            name: 'RHSFetchError',
            errorCode: 'not_connected',
            message: 'Jira account is not connected',
            statusCode: 401,
        });
        await expect(doFetchJSON('/plugins/jira/api/v2/rhs/issues', {method: 'get'})).rejects.toBeInstanceOf(RHSFetchError);
    });

    test('doFetchJSON falls back to text when the body is not JSON', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 502,
            text: () => Promise.resolve('<html>bad gateway</html>'),
        }));

        try {
            await doFetchJSON('/plugins/jira/api/v2/rhs/issues', {method: 'get'});
            throw new Error('expected doFetchJSON to reject');
        } catch (error) {
            expect(error).toBeInstanceOf(RHSFetchError);
            expect(error).not.toBeInstanceOf(SyntaxError);
            expect((error as RHSFetchError).errorCode).toBe('internal_error');
            expect((error as RHSFetchError).message).toBe('<html>bad gateway</html>');
        }
    });

    test('doFetchJSON falls back to text for checkAuth plain 401', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 401,
            text: () => Promise.resolve('Not authorized\n'),
        }));

        try {
            await doFetchJSON('/plugins/jira/api/v2/rhs/issues', {method: 'get'});
            throw new Error('expected doFetchJSON to reject');
        } catch (error) {
            expect(error).toBeInstanceOf(RHSFetchError);
            expect((error as RHSFetchError).errorCode).toBe('internal_error');
            expect((error as RHSFetchError).message).toContain('Not authorized');
        }
    });

    test('doFetchWithResponse still throws ClientError with the raw non-JSON body', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: false,
            status: 502,
            text: () => Promise.resolve('<html>bad gateway</html>'),
        }));

        try {
            await doFetchWithResponse('/plugins/jira/api/v2/rhs/issues', {method: 'get'});
            throw new Error('expected doFetchWithResponse to reject');
        } catch (error) {
            expect(error).toBeInstanceOf(ClientError);
            expect(error).not.toBeInstanceOf(RHSFetchError);
            expect((error as ClientError).message).toContain('<html>bad gateway</html>');
        }
    });

    test('getRHSIssues calls /plugins path with snake_case query params', async () => {
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
        expect(url).toContain('tab_kind=category');
        expect(url).toContain('tab_key=indeterminate');
        expect(url).toContain('sort=created');
        expect(url).not.toContain('next_page_token');
        expect(url).not.toContain('/api/v2/api/v2');
    });

    test('getRHSIssues omits empty tab_key and tab_id for assigned', async () => {
        fetchMock.mockImplementation(() => Promise.resolve({
            ok: true,
            json: () => Promise.resolve(emptyIssues),
        }));

        await getRHSIssues('/plugins/jira', {
            instanceID: 'https://x.atlassian.net',
            tab: {kind: 'assigned', name: 'Assigned'},
            sort: 'updated',
        });

        const url = fetchMock.mock.calls[0][0] as string;
        expect(url).toContain('tab_kind=assigned');
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
