// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginId} from '../support/constants';
import {expect, test} from '../support/fixture';

/**
 * @objective Verify the plugin server starts after install.
 */
test('reports plugin status as running after install', {tag: '@jira_plugin'}, async ({jira}) => {
    // # Fetch plugin statuses as admin
    const statuses = await jira.adminClient.getPluginStatuses();
    const status = statuses.find((s) => s.plugin_id === pluginId);

    // * Verify the plugin is running
    expect(status?.state).toBe(2);
});

/**
 * @objective Verify the plugin webapp bundle registers without errors.
 */
test('loads the plugin webapp bundle without errors', {tag: '@jira_plugin'}, async ({pw, jira}) => {
    // # Log in and collect plugin errors while opening a channel
    const {page, channelsPage} = await pw.testBrowser.login(jira.user);
    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(`pageerror: ${err.message}`));
    page.on('console', (msg) => {
        if (msg.type() === 'error' && msg.text().toLowerCase().includes(pluginId)) {
            errors.push(`console: ${msg.text()}`);
        }
    });
    await channelsPage.goto(jira.team.name, 'town-square');
    await page.waitForFunction((id) => Boolean((window as any).plugins?.[id]), pluginId);

    // * Verify the plugin logged no errors
    expect(errors).toEqual([]);
});

/**
 * @objective Verify the /jira slash command offers its subcommands in autocomplete.
 */
test('suggests /jira subcommands in message box autocomplete', {tag: '@jira_plugin'}, async ({pw, jira}) => {
    // # Open a channel and type the command
    const {channelsPage} = await pw.testBrowser.login(jira.user);
    await channelsPage.goto(jira.team.name, 'town-square');
    const {postCreate} = channelsPage.centerView;
    await postCreate.writeMessage('/jira ');

    // * Verify the suggestions include connect
    await expect(postCreate.suggestionList).toContainText('connect');
});
