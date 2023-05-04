// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ***************************************************************
// - [#] indicates a test step (e.g. # Go to a page)
// - [*] indicates an assertion (e.g. * Check the title)
// ***************************************************************

import {expect, test} from '@e2e-support/test_fixture';

import JiraSetupFlow from '../support/components/jira_setup_flow';
import {JIRA_SITE_URL, JIRA_EMAIL, JIRA_PASSWORD, JIRA_CLIENT_ID, JIRA_CLIENT_SECRET} from '../support/creds';
import {DEFAULT_WAIT_MILLIS} from '../support/utils';

import '../support/init_test';
import JiraSiteAuthFlow from '../support/components/jira_site_auth_flow';
import JiraInteractiveDialog from '../support/components/jira_interactive_dialog';

const pluginId = 'jira';
const slashCommand = '/' + pluginId;
const botUsername = pluginId;

test('/jira setup', async ({pw, pages, page: originalPage, context}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);
    await originalPage.close();

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    const setupFlow = new JiraSetupFlow(page, c);
    const dialog = new JiraInteractiveDialog(page);

    // # Run setup command
    await c.postMessage(`${slashCommand} setup`);
    await c.sendMessage();

    // # Go to bot DM channel
    const teamName = page.url().split('/')[3];
    await c.goto(teamName, `messages/@${botUsername}`);

    // # Go through prompts of setup flow
    await setupFlow.clickFlowChoices([
        'Continue',
        "I'll do it myself",
        'Jira Cloud (OAuth 2.0)',
    ]);

    // # Fill out interactive dialog for Jira organization name
    await dialog.fillTextField('url', JIRA_SITE_URL);
    await dialog.submit();

    await page.waitForTimeout(DEFAULT_WAIT_MILLIS);

    await setupFlow.clickFlowChoices(['Configure']);

    // # Fill out interactive dialog for client id and client secret
    await dialog.fillTextField('client_id', JIRA_CLIENT_ID);
    await dialog.fillTextField('client_secret', JIRA_CLIENT_SECRET);
    await dialog.submit();

    await setupFlow.clickFlowChoices([
        'Continue',
        'View webhook URL',
    ]);

    // * Assert webhook URL is present and correct
    const href = await dialog.getWebhookURL();
    expect(href).toMatch(/http:\/\/localhost:8065\/plugins\/jira\/instance\/.*\/api\/v2\/webhook\?secret=/);
    await dialog.submit();

    await page.waitForTimeout(1000);

    // # Trigger Jira site connect flow
    const pagePromise = page.waitForEvent('popup');
    await setupFlow.clickConnectLink();

    const jiraPage = await pagePromise;
    const authFlow = new JiraSiteAuthFlow(jiraPage);
    await jiraPage.waitForLoadState();

    // # Fill out Jira login form
    await authFlow.login(JIRA_EMAIL, JIRA_PASSWORD);

    // * Assert successful connection
    await expect(page.getByText('You\'ve successfully connected your Mattermost user account to Jira.')).toBeVisible();
});

test('/jira connect', async ({pw, pages, page: originalPage, context}) => {
    // # Log in
    const {adminUser} = await pw.getAdminClient();
    const {page} = await pw.testBrowser.login(adminUser);
    await originalPage.close();

    // # Navigate to Channels
    const c = new pages.ChannelsPage(page);
    await c.goto();

    // # Run disconnect command
    await c.postMessage(`${slashCommand} disconnect`);
    await c.sendMessage();

    await page.waitForTimeout(500);
    let post = await c.getLastPost();
    let postText = await post.container.innerText();
    expect(postText).toMatch('You have successfully disconnected your Jira account');

    // # Trigger Jira site connect flow
    const pagePromise = page.waitForEvent('popup');

    await page.waitForTimeout(500);

    // # Run connect command
    await c.postMessage(`${slashCommand} connect`);
    await c.sendMessage();

    const jiraPage = await pagePromise;
    const authFlow = new JiraSiteAuthFlow(jiraPage);
    await jiraPage.waitForLoadState();

    // # Fill out Jira login form
    await authFlow.login(JIRA_EMAIL, JIRA_PASSWORD);

    await page.waitForTimeout(3000);
    post = await c.getLastPost();
    postText = await post.container.innerText();
    expect(postText).toMatch('You have successfully connected your Jira account. Type in /jira to get started.');
});
