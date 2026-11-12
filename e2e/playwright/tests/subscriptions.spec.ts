// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {PlaywrightExtended} from '@mattermost/playwright-lib';

import {fakeCloudUser, fakeProject} from '../fake-jira/state';
import {jiraFlavor, jiraInstanceURL, mattermostURL, pluginId} from '../support/constants';
import {expect, test, type JiraFixture} from '../support/fixture';
import {connectUserViaKV, loadWebhookFixture, sendSubscriptionWebhook} from '../support/jira';
import {openChannel, pickOption} from '../support/ui';

type SubscriptionFilters = {
    events: string[];
    projects: string[];
    issue_types: string[];
    fields: Array<{key: string; inclusion: string; values: string[]}>;
};

type ChannelSubscription = {id: string; channel_id: string; name: string; filters: SubscriptionFilters};

const taskType = '10001';
const bugType = '10002';

async function subscriptionsAPI(jira: JiraFixture, method: string, path: string, body?: unknown) {
    const res = await fetch(`${mattermostURL}/plugins/${pluginId}/api/v2/subscriptions/channel${path}`, {
        method,
        headers: {Authorization: `Bearer ${jira.userClient.getToken()}`},
        body: body === undefined ? undefined : JSON.stringify(body),
    });
    expect(res.status, await res.clone().text()).toBe(200);
    return res.json();
}

function createSubscription(jira: JiraFixture, channelId: string, name: string, filters: Partial<SubscriptionFilters>): Promise<ChannelSubscription> {
    return subscriptionsAPI(jira, 'POST', '', {
        channel_id: channelId,
        instance_id: jiraInstanceURL,
        name,
        filters: {events: ['event_updated_assignee'], projects: [fakeProject.key], issue_types: [taskType], fields: [], ...filters},
    });
}

function channelSubscriptions(jira: JiraFixture, channelId: string): Promise<ChannelSubscription[]> {
    return subscriptionsAPI(jira, 'GET', `/${channelId}?${new URLSearchParams({instance_id: jiraInstanceURL})}`);
}

async function channelMessages(jira: JiraFixture, channelId: string) {
    const {order, posts} = await jira.userClient.getPosts(channelId);
    return order.map((id) => posts[id].message);
}

async function openSubscriptions(pw: PlaywrightExtended, jira: JiraFixture) {
    const {page, channelsPage} = await openChannel(pw, jira);
    await channelsPage.postMessage('/jira subscribe');
    return {page, modal: page.getByRole('dialog', {name: /Jira Subscription/})};
}

// The fixtures belong to project TES, which fake-jira doesn't serve, so the issue is moved into its project.
async function assignedWebhook(issueTypeId: string) {
    const payload = JSON.parse(await loadWebhookFixture('webhook-issue-updated-assigned.json'));
    payload.issue.fields.project.key = fakeProject.key;
    payload.issue.fields.issuetype.id = issueTypeId;
    return JSON.stringify(payload);
}

test.describe('channel subscriptions', () => {
    test.skip(jiraFlavor === 'server', 'fake-jira only serves the Cloud project APIs behind the modal');

    test.beforeEach(async ({jira}) => {
        await jira.adminClient.updateUserRoles(jira.user.id, 'system_user system_admin');
        await connectUserViaKV(jira.user.id);
    });

    /**
     * @objective Verify a channel subscription with events, issue type and field filters is saved from the subscriptions modal.
     */
    test('creates channel subscription with filters', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Start a subscription, name it and choose a project
        const {page, modal} = await openSubscriptions(pw, jira);
        await modal.getByRole('button', {name: 'Create Subscription'}).click();
        await modal.getByPlaceholder('Name').fill('Checkout bugs');
        await pickOption(page, modal.getByRole('combobox', {name: 'Project'}), fakeProject.name);

        // # Choose events, an issue type and a priority filter, then save
        await pickOption(page, modal.getByRole('combobox', {name: 'Events'}), 'Issue Created');
        await pickOption(page, modal.getByRole('combobox', {name: 'Issue Type'}), 'Bug');
        await modal.getByRole('button', {name: 'Add Filter'}).click();
        await pickOption(page, modal.locator('#fieldtype'), 'Priority');
        await pickOption(page, modal.locator('#values'), 'High');
        await modal.getByRole('button', {name: 'Add Subscription'}).click();

        // * Verify the subscription is listed
        await expect(modal.getByRole('row', {name: /Checkout bugs/})).toContainText(fakeProject.name);

        // * Verify the subscription is stored with the chosen filters
        const [saved] = await channelSubscriptions(jira, jira.channel.id);
        expect(saved).toMatchObject({
            name: 'Checkout bugs',
            filters: {events: ['event_created'], projects: [fakeProject.key], issue_types: [bugType], fields: [{key: 'priority', inclusion: 'include_any', values: ['2']}]},
        });

        // * Verify the channel is told about the new subscription
        expect(await channelMessages(jira, jira.channel.id)).toContain(`Jira subscription, "Checkout bugs", was added to this channel by ${fakeCloudUser.displayName}`);
    });

    /**
     * @objective Verify an existing channel subscription can be renamed and given more events from the subscriptions modal.
     */
    test('edits channel subscription', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Save a subscription and open it for editing
        const created = await createSubscription(jira, jira.channel.id, 'Assignee changes', {});
        const {page, modal} = await openSubscriptions(pw, jira);
        await modal.getByRole('row', {name: /Assignee changes/}).getByRole('button', {name: 'Edit'}).click();

        // # Rename it, add an event and save
        await modal.getByPlaceholder('Name').fill('Assignee and status changes');
        await pickOption(page, modal.getByRole('combobox', {name: 'Events'}), 'Issue Updated: Status');
        await modal.getByRole('button', {name: 'Save Subscription'}).click();

        // * Verify the renamed subscription is listed
        await expect(modal.getByRole('row', {name: /Assignee and status changes/})).toBeVisible();

        // * Verify the stored subscription keeps its ID and has the new name and events
        const [saved] = await channelSubscriptions(jira, jira.channel.id);
        expect(saved.id).toBe(created.id);
        expect(saved.name).toBe('Assignee and status changes');
        expect(saved.filters.events.sort()).toEqual(['event_updated_assignee', 'event_updated_status']);

        // * Verify the channel is told about the update
        expect(await channelMessages(jira, jira.channel.id)).toContain(`Jira subscription, "Assignee and status changes", was updated by ${fakeCloudUser.displayName}`);
    });

    /**
     * @objective Verify a channel subscription is removed after confirming its deletion in the subscriptions modal.
     */
    test('deletes channel subscription', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Save a subscription, then delete it from the list
        await createSubscription(jira, jira.channel.id, 'Assignee changes', {});
        const {page, modal} = await openSubscriptions(pw, jira);
        await modal.getByRole('row', {name: /Assignee changes/}).getByRole('button', {name: 'Delete'}).click();

        // # Confirm the deletion
        const confirm = page.getByRole('dialog', {name: 'Subscription', exact: true});
        await expect(confirm).toContainText('Are you sure to delete the subscription');
        await confirm.getByRole('button', {name: 'Delete'}).click();

        // * Verify the list is empty and nothing is stored
        await expect(modal.getByRole('tabpanel')).toHaveText('No subscriptions yet');
        expect(await channelSubscriptions(jira, jira.channel.id)).toEqual([]);

        // * Verify the channel is told about the removal
        await expect.poll(() => channelMessages(jira, jira.channel.id)).
            toContain(`Jira subscription, "Assignee changes", was removed from this channel by ${fakeCloudUser.displayName}`);
    });

    /**
     * @objective Verify the subscriptions modal closes with Escape and with a click on the backdrop, and asks before discarding edits.
     */
    test('closes subscriptions modal with Escape and backdrop', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Open the modal and press Escape
        const {page, channelsPage} = await openChannel(pw, jira);
        const modal = page.getByRole('dialog', {name: /Jira Subscription/});
        await channelsPage.postMessage('/jira subscribe');
        await expect(modal).toBeVisible();
        await page.keyboard.press('Escape');

        // * Verify the modal closes
        await expect(modal).toBeHidden();

        // # Reopen it, start a subscription and type a name, then click the backdrop
        await channelsPage.postMessage('/jira subscribe');
        await modal.getByRole('button', {name: 'Create Subscription'}).click();
        await modal.getByPlaceholder('Name').fill('Unsaved');
        await page.mouse.click(5, 5);

        // * Verify discarding is confirmed before the modal closes
        const confirm = page.getByRole('dialog', {name: 'Subscription', exact: true});
        await confirm.getByRole('button', {name: 'Discard'}).click();
        await expect(modal).toBeHidden();
    });
});

test.describe('subscription webhook', {tag: '@synthetic-inbound'}, () => {
    test.skip(jiraFlavor === 'server', 'subscriptions are validated against fake-jira\'s Cloud project');

    /**
     * @objective Verify a subscription webhook is posted only to channels whose subscription filters match the issue.
     */
    test('posts subscription events only to channels with matching filters', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Subscribe one channel to bugs only and another to all issue types
        await jira.adminClient.updateUserRoles(jira.user.id, 'system_user system_admin');
        await connectUserViaKV(jira.user.id);
        const allTypes = await jira.adminClient.createChannel(pw.random.channel({teamId: jira.team.id, name: 'all-types', displayName: 'All types'}));
        await jira.adminClient.addToChannel(jira.user.id, allTypes.id);
        await createSubscription(jira, jira.channel.id, 'Bugs only', {issue_types: [bugType]});
        await createSubscription(jira, allTypes.id, 'All types', {issue_types: [taskType, bugType]});
        const assigned = /\*\*assigned\*\* .+ to/;

        // # Deliver an assignee change for a task
        const res = await sendSubscriptionWebhook(await assignedWebhook(taskType));
        expect(res.status).toBe(200);

        // * Verify it reaches the channel subscribed to all types, but not the bugs-only one
        await expect.poll(() => channelMessages(jira, allTypes.id)).toContainEqual(expect.stringMatching(assigned));
        expect(await channelMessages(jira, jira.channel.id)).not.toContainEqual(expect.stringMatching(assigned));

        // # Deliver an assignee change for a bug
        await sendSubscriptionWebhook(await assignedWebhook(bugType));

        // * Verify it reaches the bugs-only channel
        await expect.poll(() => channelMessages(jira, jira.channel.id)).toContainEqual(expect.stringMatching(assigned));
    });
});
