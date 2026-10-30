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

test(`user connects through the ${jiraFlavor} OAuth round-trip`, async ({pw, jira}) => {
    const {page} = await pw.testBrowser.login(jira.user);

    await connectUserViaBrowser(page);
    await expect(page.getByText(`Jira account: ${expectedAccount}`)).toBeVisible();

    const calls = (await fakeJira.requests()).map((r) => `${r.method} ${r.path}`);
    expect(calls).toEqual(expect.arrayContaining(expectedHandshake));

    const info = await page.request.get(`${mattermostURL}/plugins/${pluginId}/api/v2/userinfo`);
    expect(await info.json()).toMatchObject({is_connected: true});
});

test('a KV-seeded connection is recognized without an OAuth round-trip', async ({jira}) => {
    await connectUserViaKV(jira.user.id);

    const info = await fetch(`${mattermostURL}/plugins/${pluginId}/api/v2/userinfo`, {
        headers: {Authorization: `Bearer ${jira.userClient.getToken()}`},
    });
    expect(await info.json()).toMatchObject({is_connected: true});
    expect(await fakeJira.requests()).toEqual([]);
});

test('connect fails gracefully when Jira rejects the user lookup', async ({pw, jira}) => {
    await fakeJira.fault('GET', '/rest/api/2/myself', 500);
    const {page} = await pw.testBrowser.login(jira.user);

    await page.goto(`${mattermostURL}/plugins/${pluginId}/user/start`);

    await expect(page.getByText(expectedConnectFailure)).toBeVisible();
    const info = await page.request.get(`${mattermostURL}/plugins/${pluginId}/api/v2/userinfo`);
    expect(await info.json()).toMatchObject({is_connected: false});
});
