// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import fs from 'node:fs';
import path from 'node:path';

import type {CloudSite} from '../fake-jira/state';

// docker compose reads the same file, so MM_PORT/MM_IMAGE stay in sync between the stack and the tests.
const envFile = path.join(__dirname, '../.env');
if (fs.existsSync(envFile)) {
    process.loadEnvFile(envFile);
}

export const pluginId = 'jira';
export const pluginDistDir = path.join(__dirname, '../../../dist');
export const serverTestdataDir = path.join(__dirname, '../../../server/testdata');

// Must match the server's SiteURL; the plugin rejects "localhost" so 127.0.0.1 is used.
process.env.PW_BASE_URL ??= `http://127.0.0.1:${process.env.MM_PORT ?? 8065}`;
export const mattermostURL = process.env.PW_BASE_URL;

export type JiraFlavor = 'cloud' | 'server';
export const jiraFlavor: JiraFlavor = process.env.JIRA_FLAVOR === 'server' ? 'server' : 'cloud';

// Fake hostnames resolve from the Mattermost container (extra_hosts) and from Chromium
// (--host-resolver-rules), since the OAuth flows bounce the browser through them.
export const fakeJiraHost = 'fake-jira';
export const fakeCloudSiteHost = 'e2e.atlassian.net';
export const fakeJiraPort = Number(process.env.FAKE_JIRA_PORT ?? 8090);
export const fakeJiraURL = `http://${fakeJiraHost}:${fakeJiraPort}`;
export const fakeJiraControlURL = `http://localhost:${fakeJiraPort}`;

export const fakeCloudSite: CloudSite = {
    cloudId: 'e2e-cloud-id',
    siteURL: `http://${fakeCloudSiteHost}:${fakeJiraPort}`,
    clientId: 'e2e-oauth-client-id',
    clientSecret: 'e2e-oauth-client-secret',
};

export const jiraInstanceURL = jiraFlavor === 'cloud' ? fakeCloudSite.siteURL : fakeJiraURL;

// The fake and real-Jira suites share one stack, and a second installed instance makes /user/start ask which one to use.
export const knownCloudSiteURLs = [fakeCloudSite.siteURL, process.env.JIRA_CLOUD_URL?.replace(/\/$/, '')].filter((url): url is string => Boolean(url));

export const webhookSecret = 'e2e-webhook-secret';
export const encryptionKey = 'e2e-encryption-key-0123456789abc';
