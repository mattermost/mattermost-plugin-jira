// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-console */

import {test} from '@playwright/test';

import {saveSession, signInToAtlassian} from '../real-jira/atlassian_session';
import {realJira} from '../real-jira/env';

// Forces a fresh Atlassian session; the real-Jira suite also refreshes it by itself once it's 3 days old.
test('sign in to Atlassian as the regular test user', async ({page}) => {
    test.setTimeout(3 * 60_000);
    await signInToAtlassian(page, 2 * 60_000);
    await saveSession(page);
    console.log(`Saved the Atlassian session to ${realJira.userSessionPath}`);
});
