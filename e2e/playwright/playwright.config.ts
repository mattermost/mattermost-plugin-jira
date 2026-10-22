// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-process-env */

import {defineConfig, devices} from '@playwright/test';

import {fakeCloudSiteHost, fakeJiraHost, mattermostURL} from './support/constants';
import type {FailureScreenshotOptions} from './support/fixture';

const isCI = Boolean(process.env.CI);

export default defineConfig<FailureScreenshotOptions>({
    testDir: './tests',
    outputDir: './test-results',
    globalSetup: require.resolve('./global_setup'),
    fullyParallel: false,
    workers: 1,
    forbidOnly: isCI,
    retries: isCI ? 1 : 0,
    timeout: 60_000,
    expect: {timeout: 15_000},
    reporter: isCI ? [['github'], ['html', {open: 'never', outputFolder: 'playwright-report'}]] : [['list'], ['html', {open: 'never'}]],
    use: {
        baseURL: mattermostURL,
        headless: process.env.PW_HEADLESS !== 'false',

        // Only fake-Jira tokens and a throwaway server's sessions end up in these, so CI keeps them too.
        trace: 'retain-on-failure',
        screenshot: 'off',
        failureScreenshots: true,
        launchOptions: {
            args: [`--host-resolver-rules=MAP ${fakeJiraHost} 127.0.0.1, MAP ${fakeCloudSiteHost} 127.0.0.1`],
        },
    },
    projects: [
        {
            name: 'setup',
            testMatch: /setup\/.*\.setup\.ts/,
            use: {...devices['Desktop Chrome']},
        },
        {
            name: 'chrome',
            testIgnore: /setup\//,
            dependencies: ['setup'],
            use: {...devices['Desktop Chrome'], viewport: {width: 1280, height: 1024}},
        },
    ],
});
