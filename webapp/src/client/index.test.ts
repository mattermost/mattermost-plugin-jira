// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ClientError} from '@mattermost/client';

import {doFetchWithResponse} from './index';

const fetchMock = global.fetch as jest.Mock;

describe('client', () => {
    beforeEach(() => {
        fetchMock.mockReset();
    });

    afterEach(() => {
        fetchMock.mockReset();
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
            expect((error as ClientError).message).toContain('<html>bad gateway</html>');
        }
    });
});
