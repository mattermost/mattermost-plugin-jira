// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {defineConfig, devices} from '@playwright/test';

import {mattermostURL} from './support/constants';
import type {FailureScreenshotOptions} from './support/fixture';

const isCI = Boolean(process.env.CI);

// Full journeys against a real Jira Cloud test site. Needs a normal `make dist` build, not `make e2e-dist`.
export default defineConfig<FailureScreenshotOptions>({
    testDir: './real-jira',
    outputDir: './test-results/real-jira',
    globalSetup: require.resolve('./real-jira/global_setup'),
    fullyParallel: false,
    workers: 1,
    forbidOnly: isCI,
    retries: 0,
    timeout: 120_000,
    expect: {timeout: 30_000},
    reporter: [['list'], ['html', {open: 'never', outputFolder: 'playwright-report/real-jira'}]],
    use: {
        baseURL: mattermostURL,

        // Visible by default locally; PW_HEADLESS=true hides it.
        headless: isCI || process.env.PW_HEADLESS === 'true',
        launchOptions: {slowMo: Number(process.env.PW_SLOWMO ?? 0)},

        // Real tokens and Atlassian cookies end up in traces, so they stay local.
        trace: isCI ? 'off' : 'retain-on-failure',
        screenshot: 'off',
        failureScreenshots: !isCI,
    },
    projects: [
        {
            name: 'real-setup',
            testMatch: /instance\.setup\.ts$/,
            use: {...devices['Desktop Chrome']},
        },
        {
            name: 'real-jira',
            testMatch: /\.spec\.ts$/,
            dependencies: ['real-setup'],
            use: {...devices['Desktop Chrome'], viewport: {width: 1280, height: 1024}},
        },
    ],
});
