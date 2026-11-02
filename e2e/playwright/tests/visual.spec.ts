// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Locator, Page} from '@playwright/test';

import {fakeProject} from '../fake-jira/state';
import {expect, test, type JiraFixture} from '../support/fixture';
import {connectUserViaKV, loadWebhookFixture, sendLegacyWebhook} from '../support/jira';

test.describe('visual', {tag: '@visual'}, () => {
    // eslint-disable-next-line no-process-env
    test.skip(!process.env.PW_VISUAL_BROWSER_WS, 'baselines only match the dockerized browser (npm run test:visual)');

    // Core UI around the target (avatars, status, timestamps) changes between runs.
    async function settle(pw: PlaywrightExtended, target: Locator) {
        const page = target.page();
        await expect(page.locator('.LoadingScreen')).toBeHidden();
        await pw.hideDynamicChannelsContent(page);
        await page.mouse.move(0, 0);
        await page.evaluate(() => document.fonts.ready);

        // A modal measured mid slide-in gets snapped to the wrong pixel.
        await page.evaluate(() => Promise.all(document.getAnimations().
            filter((a) => a.effect?.getTiming().iterations !== Infinity).
            map((a) => a.finished.catch(() => undefined))));

        // Bottom-anchored posts and centred modals can sit at a fractional y, which moves text by a pixel row.
        await target.evaluate((el: HTMLElement) => {
            const {x, y} = el.getBoundingClientRect();
            el.style.transform = `translate(${Math.round(x) - x}px, ${Math.round(y) - y}px)`;
        });
    }

    async function openChannel(pw: PlaywrightExtended, jira: JiraFixture) {
        const {page, channelsPage} = await pw.testBrowser.login(jira.user);
        await channelsPage.goto(jira.team.name, jira.channel.name);
        await channelsPage.toBeVisible();
        return {page, channelsPage};
    }

    async function openPostAction(pw: PlaywrightExtended, jira: JiraFixture, action: string) {
        await connectUserViaKV(jira.user.id);
        const created = await jira.userClient.createPost({channel_id: jira.channel.id, message: 'Checkout button overlaps the cart total'});
        const {page, channelsPage} = await openChannel(pw, jira);
        const post = await channelsPage.centerView.getPostById(created.id);
        await post.hover();
        await post.postMenu.actionsButton.click();
        await page.getByRole('menuitem', {name: action}).click();
        return page;
    }

    async function pickOption(page: Page, field: Locator, option: string) {
        await field.click();
        await page.getByRole('option', {name: option, exact: true}).click();
    }

    /**
     * @objective Verify the issue created post renders like the baseline.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for issue created post', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Deliver an issue created webhook and find its post
        const res = await sendLegacyWebhook(await loadWebhookFixture('webhook-issue-created.json'), {team: jira.team.name, channel: jira.channel.name});
        expect(res.status).toBe(200);
        let postId: string | undefined;
        await expect.poll(async () => {
            const {order, posts} = await jira.userClient.getPosts(jira.channel.id);
            postId = order.find((id) => JSON.stringify(posts[id].props?.attachments ?? '').includes('TES-41'));
            return postId;
        }).toBeTruthy();

        // # Open the channel
        const {channelsPage} = await openChannel(pw, jira);
        const post = await channelsPage.centerView.getPostById(postId!);
        await settle(pw, post.body);

        // * Verify the post matches the baseline
        await expect(post.body).toHaveScreenshot('issue-created-post.png');
    });

    /**
     * @objective Verify the create issue modal renders like the baseline, before and after choosing a project.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for create issue modal', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Open Create Jira Issue from a post's message actions
        const page = await openPostAction(pw, jira, 'Create Jira Issue');
        const dialog = page.getByRole('dialog', {name: 'Create Jira Issue'});
        const modal = dialog.getByRole('document');
        await expect(dialog.getByRole('combobox', {name: 'Project'})).toBeVisible();
        await settle(pw, modal);

        // * Verify the empty modal matches the baseline
        await expect(modal).toHaveScreenshot('create-issue-empty.png');

        // # Choose a project and issue type
        await pickOption(page, dialog.getByRole('combobox', {name: 'Project'}), fakeProject.name);
        await pickOption(page, dialog.getByRole('combobox', {name: 'Issue Type'}), 'Task');
        await expect(dialog.getByRole('textbox', {name: 'Summary'})).toBeVisible();
        await settle(pw, modal);

        // * Verify the modal with the project's fields matches the baseline
        await expect(modal).toHaveScreenshot('create-issue-fields.png');
    });

    /**
     * @objective Verify the attach to issue modal renders like the baseline with search results open.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for attach to issue modal', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Open Attach to Jira Issue from a post's message actions and open the issue search
        const page = await openPostAction(pw, jira, 'Attach to Jira Issue');
        const dialog = page.getByRole('dialog', {name: 'Attach Message to Jira Issue'});
        const modal = dialog.getByRole('document');
        await dialog.getByRole('combobox', {name: 'Jira Issue'}).click();
        await expect(page.getByRole('option').first()).toBeVisible();
        await settle(pw, modal);

        // * Verify the modal matches the baseline
        await expect(modal).toHaveScreenshot('attach-to-issue.png');
    });

    /**
     * @objective Verify the subscriptions modal and its add form render like the baselines.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for subscriptions modal', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Open the subscriptions modal as a system admin
        await jira.adminClient.updateUserRoles(jira.user.id, 'system_user system_admin');
        await connectUserViaKV(jira.user.id);
        const {page, channelsPage} = await openChannel(pw, jira);
        await channelsPage.postMessage('/jira subscribe');
        const modal = page.getByRole('dialog', {name: 'Jira Subscriptions'});
        await expect(modal.getByRole('button', {name: 'Create Subscription'})).toBeVisible();
        await settle(pw, modal);

        // * Verify the empty list matches the baseline
        await expect(modal).toHaveScreenshot('subscriptions-empty.png');

        // # Start a subscription and choose a project
        await modal.getByRole('button', {name: 'Create Subscription'}).click();
        await pickOption(page, modal.getByRole('combobox', {name: 'Project'}), fakeProject.name);
        await expect(modal.getByText('Issue Type', {exact: true})).toBeVisible();
        await settle(pw, modal);

        // * Verify the add form matches the baseline
        await expect(modal).toHaveScreenshot('subscription-form.png');
    });
});
