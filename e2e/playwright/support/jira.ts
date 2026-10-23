// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fs from 'node:fs/promises';
import path from 'node:path';

import type {Client4} from '@mattermost/client';
import type {Page} from '@playwright/test';
import {expect} from '@playwright/test';

import {fakeCloudUser, fakeJiraUser} from '../fake-jira/state';

import {
    fakeCloudSite,
    fakeJiraURL,
    jiraFlavor,
    jiraInstanceURL,
    knownCloudSiteURLs,
    mattermostURL,
    pluginId,
    serverTestdataDir,
    webhookSecret,
} from './constants';
import {fakeJira} from './fake_jira';
import {seedConnection} from './kv';

export async function runCommand(client: Client4, command: string, channelId: string, teamId: string) {
    return client.executeCommand(command, {channel_id: channelId, team_id: teamId});
}

// Command responses are ephemeral posts, so state is verified through the plugin API instead.
export async function installedInstances(client: Client4): Promise<string> {
    const res = await fetch(`${mattermostURL}/plugins/${pluginId}/autocomplete/installed-instance`, {
        headers: {Authorization: `Bearer ${client.getToken()}`},
    });
    return res.text();
}

export async function uninstallInstances(adminClient: Client4, channelId: string, teamId: string) {
    await runCommand(adminClient, `/jira instance uninstall server ${fakeJiraURL}`, channelId, teamId);
    for (const siteURL of knownCloudSiteURLs) {
        await runCommand(adminClient, `/jira instance uninstall cloud-oauth ${siteURL}`, channelId, teamId);
    }
}

export async function installServerInstance(adminClient: Client4, channelId: string, teamId: string) {
    await runCommand(adminClient, `/jira instance install server ${fakeJiraURL}`, channelId, teamId);
    await expectInstanceInstalled(adminClient, fakeJiraURL);
}

// Drives the real admin wizard: the client ID/secret can only be entered through the bot's interactive dialog.
type CloudOAuthApp = {siteURL: string; clientId: string; clientSecret: string};

export async function installCloudInstance(adminClient: Client4, adminPage: Page, teamName: string, channelId: string, teamId: string, app: CloudOAuthApp = fakeCloudSite) {
    await runCommand(adminClient, `/jira instance install cloud-oauth ${app.siteURL}`, channelId, teamId);
    await expectInstanceInstalled(adminClient, app.siteURL);

    await adminPage.goto(`/${teamName}/messages/@jira`);
    await adminPage.getByRole('button', {name: 'Configure'}).last().click();

    const dialog = adminPage.getByRole('dialog', {name: /Configure your Jira Cloud OAuth 2.0/});
    await expect(dialog).toBeVisible();
    await dialog.getByLabel('Jira Cloud organization').fill(app.siteURL);
    await dialog.getByLabel('Jira OAuth Client ID').fill(app.clientId);
    await dialog.getByLabel('Jira OAuth Client Secret').fill(app.clientSecret);
    await dialog.getByRole('button', {name: 'Continue'}).click();
    await expect(dialog).toBeHidden();
}

// Matches the plugin's instancePath(): base64.URLEncoding (padded), which url.PathEscape leaves as-is.
export function instancePathSegment(instanceID: string) {
    return Buffer.from(instanceID).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
}

async function expectInstanceInstalled(client: Client4, instanceURL: string) {
    await expect.poll(() => installedInstances(client), {message: `Jira instance ${instanceURL} should be installed`}).toContain(instanceURL);
}

export async function connectUserViaBrowser(page: Page) {
    await page.goto(`${mattermostURL}/plugins/${pluginId}/user/start`);
    await expect(page.getByText(/connected to Jira/i).first()).toBeVisible();
}

// Connects a user without the OAuth flow, using tokens fake-jira will accept.
export async function connectUserViaKV(mattermostUserId: string) {
    const tokens = await fakeJira.issueTokens();
    if (jiraFlavor === 'cloud') {
        seedConnection({mattermostUserId, instanceId: jiraInstanceURL, instanceType: 'cloud-oauth', jiraAccount: fakeCloudUser, credentials: {oauth2: tokens}});
        return;
    }
    seedConnection({
        mattermostUserId,
        instanceId: jiraInstanceURL,
        instanceType: 'server',
        jiraAccount: fakeJiraUser,
        credentials: {oauth1: {token: tokens.access_token, secret: tokens.refresh_token}},
    });
}

export async function loadWebhookFixture(name: string): Promise<string> {
    return fs.readFile(path.join(serverTestdataDir, name), 'utf8');
}

type LegacyWebhookOptions = {
    team: string;
    channel: string;
    secret?: string;
    params?: Record<string, string>;
};

export async function sendLegacyWebhook(payload: string, {team, channel, secret = webhookSecret, params = {}}: LegacyWebhookOptions) {
    const query = new URLSearchParams({secret, team, channel, ...params});
    return fetch(`${mattermostURL}/plugins/${pluginId}/webhook?${query}`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: payload,
    });
}
