// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-console */

import {test} from '@playwright/test';

import {canSignInAutomatically, saveSession, signInToAtlassian} from '../real-jira/atlassian_session';
import {realJira} from '../real-jira/env';

// Saves the regular user's Atlassian session for the real-Jira suite. Signs in by itself when the password and
// TOTP secret are set; otherwise a person finishes the Microsoft login in the window.
test('sign in to Atlassian as the regular test user', async ({page}) => {
    test.setTimeout(10 * 60_000);
    if (!canSignInAutomatically()) {
        console.log(`Finish signing in as the regular test user (JIRA_CLOUD_USER_EMAIL) in the browser window (Microsoft login + MFA)...`);
    }

    await signInToAtlassian(page, 9 * 60_000);
    await saveSession(page);
    console.log(`Saved the Atlassian session to ${realJira.userSessionPath}`);
});
