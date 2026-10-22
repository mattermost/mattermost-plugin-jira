// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginId} from '../support/constants';
import {expect, test} from '../support/fixture';

test('plugin server reports Running', async ({jira}) => {
    const statuses = await jira.adminClient.getPluginStatuses();
    const status = statuses.find((s) => s.plugin_id === pluginId);

    expect(status?.state).toBe(2);
});

test('webapp bundle loads without errors', async ({pw, jira}) => {
    const {page} = await pw.testBrowser.login(jira.user);

    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(`pageerror: ${err.message}`));
    page.on('console', (msg) => {
        if (msg.type() === 'error' && msg.text().toLowerCase().includes(pluginId)) {
            errors.push(`console: ${msg.text()}`);
        }
    });

    await page.goto(`/${jira.team.name}/channels/town-square`);
    await page.waitForFunction((id) => Boolean((window as any).plugins?.[id]), pluginId);

    expect(errors).toEqual([]);
});

test('/jira slash command autocompletes', async ({pw, jira}) => {
    const {page} = await pw.testBrowser.login(jira.user);
    await page.goto(`/${jira.team.name}/channels/town-square`);

    await page.getByTestId('post_textbox').fill('/jira ');

    await expect(page.locator('#suggestionList')).toContainText('connect');
});
