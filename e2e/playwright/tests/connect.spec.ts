// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {fakeCloudSite, jiraFlavor, mattermostURL, pluginId} from '../support/constants';
import {fakeCloudUser, fakeJiraUser} from '../fake-jira/state';
import {fakeJira} from '../support/fake_jira';
import {expect, test} from '../support/fixture';
import {connectUserViaBrowser, connectUserViaKV} from '../support/jira';

const expectedHandshake = {
    server: [
        'POST /plugins/servlet/oauth/request-token',
        'GET /plugins/servlet/oauth/authorize',
        'POST /plugins/servlet/oauth/access-token',
        'GET /rest/api/2/myself',
    ],
    // accessible-resources is only called until the instance caches its cloud ID, so it's not asserted here;
    // the gateway path proves the ID was resolved.
    cloud: [
        'GET /authorize',
        'POST /oauth/token',
        `GET /ex/jira/${fakeCloudSite.cloudId}/rest/api/2/myself`,
    ],
}[jiraFlavor];

const expectedAccount = jiraFlavor === 'cloud' ? fakeCloudUser.displayName : `${fakeJiraUser.displayName} (${fakeJiraUser.name})`;

// Server renders the friendly template; Cloud currently surfaces the raw handler error.
const expectedConnectFailure = jiraFlavor === 'cloud' ? /Error occurred while getting Jira user/ : /Failed to connect to Jira\./;

/**
 * @objective Verify a user connects their Jira account through the full OAuth handshake.
 */
test(`connects user through the ${jiraFlavor} OAuth round-trip`, {tag: '@jira_plugin'}, async ({pw, jira}) => {
    // # Log in and start the connect flow
    const {page} = await pw.testBrowser.login(jira.user);
    await connectUserViaBrowser(page);

    // * Verify the connected Jira account is shown
    await expect(page.getByText(`Jira account: ${expectedAccount}`)).toBeVisible();

    // * Verify the plugin made the expected OAuth calls
    const calls = (await fakeJira.requests()).map((r) => `${r.method} ${r.path}`);
    expect(calls).toEqual(expect.arrayContaining(expectedHandshake));

    // * Verify the plugin reports the user as connected
    const info = await page.request.get(`${mattermostURL}/plugins/${pluginId}/api/v2/userinfo`);
    expect(await info.json()).toMatchObject({is_connected: true});
});

/**
 * @objective Verify a connection seeded into the KV store is recognized without calling Jira.
 */
test('recognizes a KV-seeded connection without an OAuth round-trip', {tag: '@jira_plugin'}, async ({jira}) => {
    // # Seed the connection records
    await connectUserViaKV(jira.user.id);

    // * Verify the plugin reports the user as connected
    const info = await fetch(`${mattermostURL}/plugins/${pluginId}/api/v2/userinfo`, {
        headers: {Authorization: `Bearer ${jira.userClient.getToken()}`},
    });
    expect(await info.json()).toMatchObject({is_connected: true});

    // * Verify Jira was not called
    expect(await fakeJira.requests()).toEqual([]);
});

/**
 * @objective Verify connect shows an error and stays disconnected when Jira rejects the user lookup.
 */
test('shows connect error when Jira rejects the user lookup', {tag: '@jira_plugin'}, async ({pw, jira}) => {
    // # Make Jira fail the user lookup, then start the connect flow
    await fakeJira.fault('GET', '/rest/api/2/myself', 500);
    const {page} = await pw.testBrowser.login(jira.user);
    await page.goto(`${mattermostURL}/plugins/${pluginId}/user/start`);

    // * Verify the error is shown
    await expect(page.getByText(expectedConnectFailure)).toBeVisible();

    // * Verify the user is not connected
    const info = await page.request.get(`${mattermostURL}/plugins/${pluginId}/api/v2/userinfo`);
    expect(await info.json()).toMatchObject({is_connected: false});
});
