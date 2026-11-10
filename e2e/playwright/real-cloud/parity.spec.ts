// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {expect, test} from '@playwright/test';

import {ContractValidator} from '../fake-jira/contract/validator';
import {assignableUsers, cloudUser, createFields, createMeta, projectSearch, projectStatuses, searchIssues} from '../fake-jira/state';
import {fakeCloudSite} from '../support/constants';

const siteURL = process.env.JIRA_CLOUD_URL?.replace(/\/$/, '');

// The admin account backs the plugin's Admin API Token; the user account is a regular Jira user.
const accounts = [
    {role: 'admin', email: process.env.JIRA_CLOUD_ADMIN_EMAIL, apiToken: process.env.JIRA_CLOUD_ADMIN_API_TOKEN},
    {role: 'user', email: process.env.JIRA_CLOUD_USER_EMAIL, apiToken: process.env.JIRA_CLOUD_USER_API_TOKEN},
];

const projectKey = process.env.JIRA_CLOUD_PROJECT_KEY || 'IT';

// Each case pairs a real Cloud endpoint with what fake-jira returns for it.
// specGap: real responses Atlassian's spec doesn't describe yet.
const cases: Array<{path: string; query?: Record<string, string>; specGap?: RegExp; fake: () => unknown}> = [
    {path: '/rest/api/2/myself', fake: () => cloudUser(fakeCloudSite)},
    {path: '/rest/api/3/myself', fake: () => cloudUser(fakeCloudSite)},
    {path: '/rest/api/3/project/search', query: {expand: 'issueTypes', maxResults: '1'}, fake: () => projectSearch(fakeCloudSite, true)},
    {path: `/rest/api/3/project/${projectKey}/statuses`, fake: () => projectStatuses(fakeCloudSite)},
    {
        path: '/rest/api/2/issue/createmeta',
        query: {projectKeys: projectKey, expand: 'projects.issuetypes.fields'},
        specGap: /issuetypes\/\d+ must NOT have additional properties \{"additionalProperty":"untranslatedName"\}/,
        // Which fields an issue type has is per-project config; summary is on every one.
        fake: () => createMeta(fakeCloudSite, {summary: createFields.summary}),
    },
    {path: '/rest/api/2/search/jql', query: {jql: `project = ${projectKey}`, maxResults: '1', fields: 'summary'}, fake: () => searchIssues(fakeCloudSite)},
    {path: '/rest/api/2/user/assignable/search', query: {project: projectKey, maxResults: '1'}, fake: () => assignableUsers(fakeCloudSite, '')},
];

const requiredVars = 'JIRA_CLOUD_URL, JIRA_CLOUD_{ADMIN,USER}_EMAIL and JIRA_CLOUD_{ADMIN,USER}_API_TOKEN';
const configured = Boolean(siteURL && accounts.every((a) => a.email && a.apiToken));
if (!configured && process.env.CI) {
    throw new Error(`${requiredVars} must be set in CI`);
}
test.skip(!configured, `${requiredVars} are not set`);

const validator = new ContractValidator();

for (const {role, email, apiToken} of accounts) {
    for (const {path, query, specGap, fake} of cases) {
        /**
         * @objective Verify fake-jira only returns fields real Jira Cloud returns, and real responses match the committed spec.
         *
         * @precondition
         * JIRA_CLOUD_URL and both test accounts' emails and API tokens are set.
         */
        test(`matches fake-jira fields for GET ${path} as ${role}`, {tag: '@jira_plugin'}, async ({request}) => {
            // # Call the real endpoint as this account
            const res = await request.get(`${siteURL}${path}`, {
                params: query,
                headers: {
                    Authorization: `Basic ${Buffer.from(`${email}:${apiToken}`).toString('base64')}`,
                    Accept: 'application/json',
                },
            });
            expect(res.status()).toBe(200);
            const real = await res.json();

            // * Verify the real response matches the spec and has every field fake-jira returns

            const violations = validator.validateResponse('GET', path, 200, real).filter((v) => !specGap?.test(v));
            expect(violations, 'real Cloud response violates the committed spec').toEqual([]);
            expect(missingFields(fake(), real), 'fake-jira returns fields real Cloud does not').toEqual([]);
        });
    }
}

// Arrays are compared by their first element, since real and fake hold different items.
function missingFields(fake: unknown, real: unknown, at = ''): string[] {
    if (Array.isArray(fake) && Array.isArray(real)) {
        return fake.length && real.length ? missingFields(fake[0], real[0], `${at}/0`) : [];
    }
    if (!isObject(fake) || !isObject(real)) {
        return [];
    }
    return Object.entries(fake).flatMap(([key, value]) => {
        const fieldPath = `${at}/${key}`;
        if (!(key in real)) {
            return [fieldPath];
        }
        return missingFields(value, real[key], fieldPath);
    });
}

function isObject(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}
