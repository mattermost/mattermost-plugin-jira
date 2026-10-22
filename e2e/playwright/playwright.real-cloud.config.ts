// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {defineConfig} from '@playwright/test';

import './support/constants';

// Talks only to Atlassian (no Mattermost): fake-jira parity checks, plus the one-off `npm run real:login`.
export default defineConfig({
    testDir: './real-cloud',
    outputDir: './test-results/real-cloud',
    workers: 1,
    forbidOnly: Boolean(process.env.CI),
    timeout: 30_000,
    reporter: process.env.CI ? [['github'], ['list']] : [['list']],
    projects: [
        {name: 'parity', testIgnore: /\.setup\.ts$/},
        {name: 'atlassian-login', testMatch: /atlassian_login\.setup\.ts$/, use: {headless: false}},
    ],
});
