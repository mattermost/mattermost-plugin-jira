// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-console */

import {baseGlobalSetup} from '@mattermost/playwright-lib';

import {loginAdmin} from '../support/admin';
import {installPlugin} from '../support/plugin';

import {jiraApi, markerJQL} from './atlassian';
import {ensureAtlassianSession} from './atlassian_session';
import {missingRealJiraEnv, realJira} from './env';

export default async function globalSetup() {
    const missing = missingRealJiraEnv();
    if (missing.length) {
        throw new Error(`Set ${missing.join(', ')} in e2e/playwright/.env (see .env.example)`);
    }
    await ensureAtlassianSession();
    await sweepLeftoverIssues();

    await baseGlobalSetup();
    const adminClient = await loginAdmin();
    await installPlugin(adminClient, {
        adminapitoken: realJira.admin.apiToken,
        adminemail: realJira.admin.email,
        rolesallowedtoeditjirasubscriptions: 'users',
    });
}

async function sweepLeftoverIssues() {
    const keys = await jiraApi.searchKeys(realJira.admin, `${markerJQL(realJira.projectKey, realJira.issueMarker)} AND created < -1h`);
    for (const key of keys) {
        await jiraApi.deleteIssue(realJira.admin, key).catch((err) => console.warn(`Could not delete leftover ${key}: ${err.message}`));
    }
    if (keys.length) {
        console.log(`Deleted ${keys.length} leftover ${realJira.issueMarker} issues from ${realJira.projectKey}`);
    }
}
