// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import path from 'node:path';

import '../support/constants';

export type JiraAccount = {email: string; apiToken: string; password?: string; totpSecret?: string};

const required = [
    'JIRA_CLOUD_URL',
    'JIRA_CLOUD_ADMIN_EMAIL',
    'JIRA_CLOUD_ADMIN_API_TOKEN',
    'JIRA_CLOUD_USER_EMAIL',
    'JIRA_CLOUD_USER_API_TOKEN',
    'JIRA_CLOUD_OAUTH_CLIENT_ID',
    'JIRA_CLOUD_OAUTH_CLIENT_SECRET',
] as const;

export function missingRealJiraEnv() {
    return required.filter((name) => !process.env[name]);
}

const env = (name: (typeof required)[number]) => process.env[name] ?? '';

export const realJira = {
    siteURL: env('JIRA_CLOUD_URL').replace(/\/$/, ''),
    projectKey: process.env.JIRA_CLOUD_PROJECT_KEY || 'IT',
    admin: {email: env('JIRA_CLOUD_ADMIN_EMAIL'), apiToken: env('JIRA_CLOUD_ADMIN_API_TOKEN')} as JiraAccount,
    user: {
        email: env('JIRA_CLOUD_USER_EMAIL'),
        apiToken: env('JIRA_CLOUD_USER_API_TOKEN'),
        password: process.env.JIRA_CLOUD_USER_PASSWORD || undefined,
        totpSecret: process.env.JIRA_CLOUD_USER_TOTP_SECRET || undefined,
    } as JiraAccount,
    oauth: {clientId: env('JIRA_CLOUD_OAUTH_CLIENT_ID'), clientSecret: env('JIRA_CLOUD_OAUTH_CLIENT_SECRET')},

    // Atlassian browser session of the regular user, saved by `npm run real:login`.
    userSessionPath: path.join(__dirname, '../.auth/atlassian-user.json'),

    // Every issue the suite creates carries this marker so cleanup can find leftovers.
    issueMarker: 'mm-e2e',
};
