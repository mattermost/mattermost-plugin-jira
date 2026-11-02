// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Client4} from '@mattermost/client';
import type {Post} from '@mattermost/types/posts';

import {expect, test} from '../support/fixture';
import {loadWebhookFixture, sendLegacyWebhook} from '../support/jira';

type SlackAttachment = {pretext?: string; text?: string};

const fixtureIssueKey = 'TES-41';

async function issuePosts(client: Client4, channelId: string): Promise<Post[]> {
    const {order, posts} = await client.getPosts(channelId);
    return order.
        map((id) => posts[id]).
        filter((post) => JSON.stringify([post.message, post.props?.attachments]).includes(fixtureIssueKey));
}

// Public channels skip the connected-member guard (DM/GM only), so no Jira connection is needed here.
test.describe('legacy webhook', {tag: '@synthetic-inbound'}, () => {
    /**
     * @objective Verify an issue created webhook is posted to the channel named in the URL.
     */
    test('posts issue created event to the channel', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Deliver an issue created webhook
        const payload = await loadWebhookFixture('webhook-issue-created.json');
        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name});
        expect(res.status).toBe(200);

        // * Verify one post with the issue summary and description
        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id)).length).toBe(1);
        const [post] = await issuePosts(jira.userClient, jira.channel.id);
        const [attachment] = post.props.attachments as SlackAttachment[];
        expect(attachment.pretext).toContain('**created** story [TES-41: Unit test summary]');
        expect(attachment.text).toBe('Unit test description, not that long');

        // # Open the channel
        const {page, channelsPage} = await pw.testBrowser.login(jira.user);
        await channelsPage.goto(jira.team.name, jira.channel.name);

        // * Verify the issue link is rendered
        await expect(page.getByRole('link', {name: 'TES-41: Unit test summary'})).toBeVisible();
    });

    /**
     * @objective Verify events not enabled in the webhook URL are skipped until they are enabled.
     */
    test('skips event types not enabled in the webhook URL', {tag: '@jira_plugin'}, async ({jira}) => {
        // # Deliver a summary update without enabling it
        const payload = await loadWebhookFixture('webhook-issue-updated-renamed.json');
        const skipped = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name});

        // * Verify nothing is posted
        expect(skipped.status).toBe(200);
        expect(await issuePosts(jira.userClient, jira.channel.id)).toHaveLength(0);

        // # Deliver it again with updated_summary enabled
        const posted = await sendLegacyWebhook(payload, {
            team: jira.team.name,
            channel: jira.channel.name,
            params: {updated_summary: '1'},
        });

        // * Verify the update is posted
        expect(posted.status).toBe(200);
        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id)).length).toBe(1);
        const [post] = await issuePosts(jira.userClient, jira.channel.id);
        expect(post.message).toContain('**updated** summary from "Unit test summary" to "Unit test summary 1"');
    });

    /**
     * @objective Verify a webhook with the wrong secret is rejected and not posted.
     */
    test('rejects webhook with the wrong secret', {tag: '@jira_plugin'}, async ({jira}) => {
        // # Deliver a webhook with a bad secret
        const payload = await loadWebhookFixture('webhook-issue-created.json');
        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name, secret: 'nope'});

        // * Verify it is forbidden and nothing is posted
        expect(res.status).toBe(403);
        expect(await issuePosts(jira.userClient, jira.channel.id)).toHaveLength(0);
    });

    /**
     * @objective Verify a webhook for a channel that does not exist is rejected.
     */
    test('rejects webhook for an unknown channel', {tag: '@jira_plugin'}, async ({jira}) => {
        // # Deliver a webhook to a missing channel
        const payload = await loadWebhookFixture('webhook-issue-created.json');
        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: 'does-not-exist'});

        // * Verify it is a bad request
        expect(res.status).toBe(400);
    });
});
