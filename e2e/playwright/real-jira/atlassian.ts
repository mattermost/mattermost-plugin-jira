// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fs from 'node:fs';

import type {BrowserContext, Page} from '@playwright/test';
import {expect} from '@playwright/test';

import {canSignInAutomatically, completeSignIn} from './atlassian_session';
import {realJira, type JiraAccount} from './env';

type IssueType = {id: string; name: string; subtask: boolean};

async function jiraRequest<T>(account: JiraAccount, method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${realJira.siteURL}${path}`, {
        method,
        headers: {
            Authorization: `Basic ${Buffer.from(`${account.email}:${account.apiToken}`).toString('base64')}`,
            Accept: 'application/json',
            'Content-Type': 'application/json',
        },
        body: body === undefined ? undefined : JSON.stringify(body),

        // Global setup calls this outside any test timeout, so a stalled request would hang the whole job.
        signal: AbortSignal.timeout(30_000),
    });
    if (!res.ok) {
        throw new Error(`Jira ${method} ${path} failed: ${res.status} ${await res.text()}`);
    }
    return (res.status === 204 ? undefined : await res.json()) as T;
}

export const jiraApi = {
    myself: (account: JiraAccount) =>
        jiraRequest<{accountId: string; displayName: string}>(account, 'GET', '/rest/api/3/myself'),

    project: (account: JiraAccount, key: string) =>
        jiraRequest<{id: string; key: string; name: string}>(account, 'GET', `/rest/api/3/project/${key}`),

    async issueTypes(account: JiraAccount, projectKey: string): Promise<IssueType[]> {
        const page = await jiraRequest<{issueTypes?: IssueType[]; values?: IssueType[]}>(account, 'GET', `/rest/api/3/issue/createmeta/${projectKey}/issuetypes`);
        return (page.issueTypes ?? page.values ?? []).filter((t) => !t.subtask);
    },

    createIssue: (account: JiraAccount, fields: {projectKey: string; issueTypeId: string; summary: string}) =>
        jiraRequest<{id: string; key: string}>(account, 'POST', '/rest/api/2/issue', {
            fields: {project: {key: fields.projectKey}, issuetype: {id: fields.issueTypeId}, summary: fields.summary},
        }),

    getIssue: (account: JiraAccount, key: string) =>
        jiraRequest<Record<string, unknown> & {id: string; key: string; fields: Record<string, unknown>}>(account, 'GET', `/rest/api/2/issue/${key}`),

    async searchKeys(account: JiraAccount, jql: string): Promise<string[]> {
        const res = await jiraRequest<{issues: Array<{key: string}>}>(account, 'POST', '/rest/api/3/search/jql', {jql, fields: ['summary'], maxResults: 100});
        return res.issues.map((i) => i.key);
    },

    deleteIssue: (account: JiraAccount, key: string) => jiraRequest<void>(account, 'DELETE', `/rest/api/3/issue/${key}`),

    addComment: (account: JiraAccount, key: string, body: string) =>
        jiraRequest<Record<string, unknown> & {id: string; body: string}>(account, 'POST', `/rest/api/2/issue/${key}/comment`, {body}),

    async commentBodies(account: JiraAccount, key: string): Promise<string[]> {
        const res = await jiraRequest<{comments: Array<{body: string}>}>(account, 'GET', `/rest/api/2/issue/${key}/comment`);
        return res.comments.map((c) => c.body);
    },
};

export function markerJQL(projectKey: string, text: string) {
    return `project = ${projectKey} AND summary ~ "\\"${text}\\""`;
}

export async function addAtlassianSession(context: BrowserContext) {
    if (!fs.existsSync(realJira.userSessionPath)) {
        throw new Error('No saved Atlassian session; run `npm run real:login` and sign in as the regular test user');
    }
    const {cookies} = JSON.parse(fs.readFileSync(realJira.userSessionPath, 'utf8'));
    await context.addCookies(cookies);
}

// Atlassian's 3LO screen: sign in if Atlassian asks, pick the site when asked, then Accept.
export async function approveAtlassianConsent(page: Page) {
    const accept = page.getByRole('button', {name: /^accept$/i});
    if (canSignInAutomatically()) {
        await completeSignIn(page, 2 * 60_000, () => accept.isVisible());
    }
    await expect(accept, 'Atlassian consent screen (if it asked to sign in, run `npm run real:login`)').toBeVisible({timeout: 30_000});

    const siteHost = new URL(realJira.siteURL).hostname;
    const sitePicker = page.getByRole('combobox');
    if (await sitePicker.count() && !(await sitePicker.first().textContent())?.includes(siteHost)) {
        await sitePicker.first().click();
        await page.getByRole('option', {name: new RegExp(siteHost.split('.')[0], 'i')}).click();
    }

    await accept.click();
}
