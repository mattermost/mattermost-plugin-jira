// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Locator, Page} from '@playwright/test';

import {fakeIssues, fakeProject} from '../fake-jira/state';
import {jiraInstanceURL, mattermostURL, pluginId} from '../support/constants';
import {fakeJira} from '../support/fake_jira';
import {expect, test, type JiraFixture} from '../support/fixture';
import {connectUserViaKV, loadWebhookFixture, sendLegacyWebhook} from '../support/jira';

test.describe('visual', {tag: '@visual'}, () => {
    // eslint-disable-next-line no-process-env
    test.skip(!process.env.PW_VISUAL_BROWSER_WS, 'baselines only match the dockerized browser (npm run test:visual)');

    // Core UI around the target (avatars, status, timestamps) changes between runs.
    async function settle(pw: PlaywrightExtended, target: Locator, {keepPointer = false} = {}) {
        const page = target.page();
        await expect(page.locator('.LoadingScreen')).toBeHidden();
        await pw.hideDynamicChannelsContent(page);
        if (!keepPointer) {
            await page.mouse.move(0, 0);
        }
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

    async function deliverWebhook(jira: JiraFixture, fixture: string, params?: Record<string, string>) {
        const {order: before} = await jira.userClient.getPosts(jira.channel.id);
        const res = await sendLegacyWebhook(await loadWebhookFixture(fixture), {team: jira.team.name, channel: jira.channel.name, params});
        expect(res.status).toBe(200);
        let postId: string | undefined;
        await expect.poll(async () => {
            const {order} = await jira.userClient.getPosts(jira.channel.id);
            postId = order.find((id) => !before.includes(id));
            return postId;
        }).toBeTruthy();
        return postId!;
    }

    async function hoverIssueLink(pw: PlaywrightExtended, jira: JiraFixture) {
        await connectUserViaKV(jira.user.id);
        const link = `${jiraInstanceURL}/browse/${fakeIssues[0].key}`;
        await jira.userClient.createPost({channel_id: jira.channel.id, message: `Can someone pick up ${link} today?`});
        const {page} = await openChannel(pw, jira);
        await page.getByRole('link', {name: link}).hover();
        return page.locator('.jira-issue-tooltip');
    }

    async function openSubscriptions(pw: PlaywrightExtended, jira: JiraFixture) {
        await jira.adminClient.updateUserRoles(jira.user.id, 'system_user system_admin');
        await connectUserViaKV(jira.user.id);
        const {page, channelsPage} = await openChannel(pw, jira);
        await channelsPage.postMessage('/jira subscribe');
        return {page, modal: page.getByRole('dialog', {name: 'Jira Subscriptions'})};
    }

    /**
     * @objective Verify the issue created post renders like the baseline.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for issue created post', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Deliver an issue created webhook
        const postId = await deliverWebhook(jira, 'webhook-issue-created.json');

        // # Open the channel
        const {channelsPage} = await openChannel(pw, jira);
        const post = await channelsPage.centerView.getPostById(postId);
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
        const {page, modal} = await openSubscriptions(pw, jira);
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

    /**
     * @objective Verify the issue comment created post renders like the baseline.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for issue comment created post', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Deliver a comment created webhook for an issue with no post in the channel
        const postId = await deliverWebhook(jira, 'webhook-cloud-comment-created.json', {updated_comments: '1'});

        // # Open the channel
        const {channelsPage} = await openChannel(pw, jira);
        const post = await channelsPage.centerView.getPostById(postId);
        await settle(pw, post.body);

        // * Verify the post matches the baseline
        await expect(post.body).toHaveScreenshot('comment-created-post.png');
    });

    /**
     * @objective Verify a comment edited webhook is threaded under the issue's post and renders like the baseline in the thread.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for issue comment reply in thread', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Deliver an issue created webhook, then a comment edited webhook for the same issue
        const rootId = await deliverWebhook(jira, 'webhook-issue-created.json');
        const replyId = await deliverWebhook(jira, 'webhook-cloud-comment-updated.json', {updated_comments: '1'});

        // * Verify the comment was threaded under the issue's post
        expect((await jira.userClient.getPost(replyId)).root_id).toBe(rootId);

        // # Open the thread
        const {channelsPage} = await openChannel(pw, jira);
        await (await channelsPage.centerView.getPostById(rootId)).openAThread();
        const reply = await channelsPage.sidebarRight.getPostById(replyId);
        await settle(pw, reply.body);

        // * Verify the reply matches the baseline
        await expect(reply.body).toHaveScreenshot('comment-thread-reply.png');
    });

    /**
     * @objective Verify an issue resolved webhook, which changes several fields at once, renders like the baseline.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for issue resolved post', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Deliver an issue resolved webhook
        const postId = await deliverWebhook(jira, 'webhook-issue-updated-resolved.json');

        // # Open the channel
        const {channelsPage} = await openChannel(pw, jira);
        const post = await channelsPage.centerView.getPostById(postId);
        await settle(pw, post.body);

        // * Verify the post matches the baseline
        await expect(post.body).toHaveScreenshot('issue-resolved-post.png');
    });

    /**
     * @objective Verify the issue card posted by /jira view, with its transition and share actions, renders like the baseline.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for /jira view issue card', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # View an issue with the slash command
        await connectUserViaKV(jira.user.id);
        const {channelsPage} = await openChannel(pw, jira);
        await channelsPage.postMessage(`/jira view ${fakeIssues[0].key}`);
        const post = await channelsPage.centerView.getLastPost();
        await expect(post.body.getByRole('button', {name: 'Share publicly'})).toBeVisible();
        await settle(pw, post.body);

        // * Verify the issue card matches the baseline
        await expect(post.body).toHaveScreenshot('issue-view-card.png');
    });

    /**
     * @objective Verify the tooltip for a Jira issue link renders the issue details like the baseline.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for issue link tooltip', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Hover a link to a Jira issue in a post
        const tooltip = await hoverIssueLink(pw, jira);
        await expect(tooltip.getByRole('heading', {name: fakeIssues[0].summary})).toBeVisible();
        await settle(pw, tooltip, {keepPointer: true});

        // * Verify the tooltip matches the baseline
        await expect(tooltip).toHaveScreenshot('issue-link-tooltip.png');
    });

    /**
     * @objective Verify the issue link tooltip shows its error state like the baseline when Jira fails the lookup.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for issue link tooltip error', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Make Jira fail the issue lookup, then hover a link to the issue
        await fakeJira.fault('GET', `/rest/api/2/issue/${fakeIssues[0].key}`, 500);
        const tooltip = await hoverIssueLink(pw, jira);
        await expect(tooltip.getByText('There was a problem loading the details for this Jira link')).toBeVisible();
        await settle(pw, tooltip, {keepPointer: true});

        // * Verify the error tooltip matches the baseline
        await expect(tooltip).toHaveScreenshot('issue-link-tooltip-error.png');
    });

    /**
     * @objective Verify the create issue modal flags a missing summary like the baseline.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for create issue required field error', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Choose a project and issue type, then submit without a summary
        const page = await openPostAction(pw, jira, 'Create Jira Issue');
        const dialog = page.getByRole('dialog', {name: 'Create Jira Issue'});
        const modal = dialog.getByRole('document');
        await pickOption(page, dialog.getByRole('combobox', {name: 'Project'}), fakeProject.name);
        await pickOption(page, dialog.getByRole('combobox', {name: 'Issue Type'}), 'Task');
        await dialog.getByRole('button', {name: 'Create'}).click();
        await expect(dialog.getByText('This field is required.')).toBeVisible();
        await settle(pw, modal);

        // * Verify the modal with the error matches the baseline
        await expect(modal).toHaveScreenshot('create-issue-required.png');
    });

    /**
     * @objective Verify the add subscription form renders like the baseline with events, issue types and a field filter chosen.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for subscription form with filters', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Start a subscription, name it and choose a project
        const {page, modal} = await openSubscriptions(pw, jira);
        await modal.getByRole('button', {name: 'Create Subscription'}).click();
        await modal.getByPlaceholder('Name').fill('Checkout bugs');
        await pickOption(page, modal.getByRole('combobox', {name: 'Project'}), fakeProject.name);

        // # Choose events, an issue type and a priority filter
        await pickOption(page, modal.getByRole('combobox', {name: 'Events'}), 'Issue Created');
        await pickOption(page, modal.getByRole('combobox', {name: 'Issue Type'}), 'Bug');
        await modal.getByRole('button', {name: 'Add Filter'}).click();
        await pickOption(page, modal.locator('#fieldtype'), 'Priority');
        await pickOption(page, modal.locator('#values'), 'High');
        await settle(pw, modal);

        // * Verify the filled form matches the baseline
        await expect(modal).toHaveScreenshot('subscription-form-filters.png');
    });

    /**
     * @objective Verify a saved subscription renders like the baselines in the subscriptions list and its edit form.
     *
     * @precondition
     * Run through `npm run test:visual`, which provides the dockerized browser the baselines come from.
     */
    test('matches baseline for saved subscription', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Save a subscription through the plugin API and open the subscriptions modal
        await jira.adminClient.updateUserRoles(jira.user.id, 'system_user system_admin');
        await connectUserViaKV(jira.user.id);
        const res = await fetch(`${mattermostURL}/plugins/${pluginId}/api/v2/subscriptions/channel`, {
            method: 'POST',
            headers: {Authorization: `Bearer ${jira.userClient.getToken()}`},
            body: JSON.stringify({
                channel_id: jira.channel.id,
                instance_id: jiraInstanceURL,
                name: 'Checkout bugs',
                filters: {
                    events: ['event_created'],
                    projects: [fakeProject.key],
                    issue_types: ['10002'],
                    fields: [{key: 'priority', inclusion: 'include_any', values: ['2']}],
                },
            }),
        });
        expect(res.status).toBe(200);
        const {modal} = await openSubscriptions(pw, jira);
        await expect(modal.getByText('Checkout bugs')).toBeVisible();
        await settle(pw, modal);

        // * Verify the list matches the baseline
        await expect(modal).toHaveScreenshot('subscriptions-list.png');

        // # Edit the subscription
        await modal.getByRole('button', {name: 'Edit'}).click();
        await expect(modal.getByText('High', {exact: true})).toBeVisible();
        await settle(pw, modal);

        // * Verify the edit form matches the baseline
        await expect(modal).toHaveScreenshot('subscription-edit.png');
    });
});
