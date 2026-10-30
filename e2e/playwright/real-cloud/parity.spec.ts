// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {expect, test} from '@playwright/test';

import {ContractValidator} from '../fake-jira/contract/validator';
import {cloudUser} from '../fake-jira/state';
import {fakeCloudSite} from '../support/constants';

const siteURL = process.env.JIRA_CLOUD_URL?.replace(/\/$/, '');

// The admin account backs the plugin's Admin API Token; the user account is a regular Jira user.
const accounts = [
    {role: 'admin', email: process.env.JIRA_CLOUD_ADMIN_EMAIL, apiToken: process.env.JIRA_CLOUD_ADMIN_API_TOKEN},
    {role: 'user', email: process.env.JIRA_CLOUD_USER_EMAIL, apiToken: process.env.JIRA_CLOUD_USER_API_TOKEN},
];

// Each case pairs a real Cloud endpoint with what fake-jira returns for it.
const cases: Array<{path: string; fake: () => unknown}> = [
    {path: '/rest/api/2/myself', fake: () => cloudUser(fakeCloudSite)},
    {path: '/rest/api/3/myself', fake: () => cloudUser(fakeCloudSite)},
];

const requiredVars = 'JIRA_CLOUD_URL, JIRA_CLOUD_{ADMIN,USER}_EMAIL and JIRA_CLOUD_{ADMIN,USER}_API_TOKEN';
const configured = Boolean(siteURL && accounts.every((a) => a.email && a.apiToken));
if (!configured && process.env.CI) {
    throw new Error(`${requiredVars} must be set in CI`);
}
test.skip(!configured, `${requiredVars} are not set`);

const validator = new ContractValidator();

for (const {role, email, apiToken} of accounts) {
    for (const {path, fake} of cases) {
        test(`[${role}] GET ${path} matches fake-jira`, async ({request}) => {
            const res = await request.get(`${siteURL}${path}`, {
                headers: {
                    Authorization: `Basic ${Buffer.from(`${email}:${apiToken}`).toString('base64')}`,
                    Accept: 'application/json',
                },
            });
            expect(res.status()).toBe(200);
            const real = await res.json();

            expect(validator.validateResponse('GET', path, 200, real), 'real Cloud response violates the committed spec').toEqual([]);
            expect(missingFields(fake(), real), 'fake-jira returns fields real Cloud does not').toEqual([]);
        });
    }
}

function missingFields(fake: unknown, real: unknown, at = ''): string[] {
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
