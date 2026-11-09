// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {JournalEntry} from '../fake-jira/state';

import {fakeJiraControlURL} from './constants';

async function control(method: string, path: string, body?: unknown) {
    const res = await fetch(`${fakeJiraControlURL}/__control${path}`, {
        method,
        headers: {'Content-Type': 'application/json'},
        body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!res.ok) {
        throw new Error(`fake-jira control ${method} ${path} failed: ${res.status} ${await res.text()}`);
    }
    return res.json();
}

export const fakeJira = {
    reset: () => control('POST', '/reset'),
    requests: (): Promise<JournalEntry[]> => control('GET', '/requests'),

    // Journaled paths keep the `/ex/jira/{cloudId}` gateway prefix, so match on the REST suffix.
    requestsTo: async (method: string, path: string): Promise<JournalEntry[]> => {
        const journal: JournalEntry[] = await control('GET', '/requests');
        return journal.filter((r) => r.method === method && r.path.endsWith(path));
    },
    contractViolations: (): Promise<string[]> => control('GET', '/contract-violations'),
    issueTokens: (): Promise<{access_token: string; refresh_token: string; expires_in: number}> => control('POST', '/tokens'),
    fault: (method: string, path: string, status: number) => control('POST', '/fault', {method, path, status}),
};
