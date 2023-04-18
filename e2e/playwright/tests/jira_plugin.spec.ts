// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import '../support/init_test';

import {
    fillTextField,
    postMessage,
    submitDialog,
    clickPostAction,
    screenshot,
    DEFAULT_WAIT_MILLIS,
} from '../support/utils';

const pluginId = 'jira';
const slashCommand = '/' + pluginId;
const botUsername = pluginId;

const TEST_CLIENT_ID = 'a'.repeat(20);
const TEST_CLIENT_SECRET = 'b'.repeat(40);

test('/jira setup', async ({pw, pages, page: originalPage}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);
    await originalPage.close();

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    // # Run setup command
    await postMessage(`${slashCommand} setup`, c, page);

    // # Go to bot DM channel
    const teamName = page.url().split('/')[3];
    await c.goto(teamName, `messages/@${botUsername}`);

    // # Go through prompts of setup flow
    let choices: string[] = [
        'Continue',
        "I'll do it myself",
        'Jira Cloud (OAuth 2.0)',
        // 'Continue',
        // 'Continue',
    ];

    let i = 0;
    for (const choice of choices) {
        i++;
        await page.waitForTimeout(DEFAULT_WAIT_MILLIS);
        await screenshot(`post_action_before_${i}`, page);
        await clickPostAction(choice, c);
        await screenshot(`post_action_after_${i}`, page);
    }

    // # Fill out interactive dialog for Jira organization name
    await fillTextField('url', 'mmtest', page);
    await submitDialog(page);

    await page.waitForTimeout(DEFAULT_WAIT_MILLIS);

    choices = [
        'Configure',
    ];

    for (const choice of choices) {
        i++;
        await page.waitForTimeout(DEFAULT_WAIT_MILLIS);
        await screenshot(`post_action_before_${i}`, page);
        await clickPostAction(choice, c);
        await screenshot(`post_action_after_${i}`, page);
    }

    // # Fill out interactive dialog for client id and client secret
    await fillTextField('client_id', TEST_CLIENT_ID, page);
    await fillTextField('client_secret', TEST_CLIENT_SECRET, page);
    await submitDialog(page);

    choices = [
        'Continue',
        'View webhook URL',
    ];

    for (const choice of choices) {
        i++;
        await page.waitForTimeout(DEFAULT_WAIT_MILLIS);
        await screenshot(`post_action_before_${i}`, page);
        await clickPostAction(choice, c);
        await screenshot(`post_action_after_${i}`, page);
    }

    const link = page.locator('#interactiveDialogModalIntroductionText a');
    const href = await link.getAttribute('href');

    expect(href).toEqual('http://localhost:8065/plugins/jira/instance/aHR0cHM6Ly9tbXRlc3QuYXRsYXNzaWFuLm5ldA==/api/v2/webhook?secret=');

    await page.close();
});
