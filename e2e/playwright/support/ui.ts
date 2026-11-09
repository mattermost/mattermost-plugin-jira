// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Locator, Page} from '@playwright/test';

import {pluginId} from './constants';
import type {JiraFixture} from './fixture';

export async function openChannel(pw: PlaywrightExtended, jira: JiraFixture) {
    const {page, channelsPage} = await pw.testBrowser.login(jira.user);

    // The webapp registers its post menu actions and slash command hook only once it has fetched the user's connection;
    // a /jira subscribe sent earlier goes to the server and gets the help text instead of the modal.
    const pluginReady = page.waitForResponse((res) => res.url().includes(`/plugins/${pluginId}/api/v2/userinfo`));
    await channelsPage.goto(jira.team.name, jira.channel.name);
    await pluginReady;
    await channelsPage.toBeVisible();
    return {page, channelsPage};
}

export async function openPostAction(pw: PlaywrightExtended, jira: JiraFixture, message: string, action: string) {
    const created = await jira.userClient.createPost({channel_id: jira.channel.id, message});
    const {page, channelsPage} = await openChannel(pw, jira);
    const post = await channelsPage.centerView.getPostById(created.id);
    await post.hover();
    await post.postMenu.actionsButton.click();
    await page.getByRole('menuitem', {name: action}).click();
    return {page, channelsPage, post: created};
}

export async function pickOption(page: Page, field: Locator, option: string) {
    await field.click();
    await page.getByRole('option', {name: option, exact: true}).click();
}

// Slash command replies are ephemeral, so they only exist in the browser that ran the command.
export async function runSlashCommand(pw: PlaywrightExtended, jira: JiraFixture, command: string) {
    const {page, channelsPage} = await openChannel(pw, jira);
    await channelsPage.postMessage(command);
    return {page, channelsPage, reply: (await channelsPage.centerView.getLastPost()).body};
}
