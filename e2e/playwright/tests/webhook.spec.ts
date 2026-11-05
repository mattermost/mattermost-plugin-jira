// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Client4} from '@mattermost/client';
import type {Post} from '@mattermost/types/posts';

import {expect, test} from '../support/fixture';
import {loadWebhookFixture, sendLegacyWebhook} from '../support/jira';

type SlackAttachment = {pretext?: string; text?: string};

const fixtureIssueKey = 'TES-41';

async function issuePosts(client: Client4, channelId: string, issueKey = fixtureIssueKey): Promise<Post[]> {
    const {order, posts} = await client.getPosts(channelId);
    return order.
        map((id) => posts[id]).
        filter((post) => JSON.stringify([post.message, post.props?.attachments]).includes(issueKey));
}

function headline(post: Post) {
    const [attachment] = (post.props?.attachments ?? []) as SlackAttachment[];
    return attachment?.pretext ?? post.message;
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
     * @objective Verify a comment created webhook is posted as a reply to the issue's existing post.
     */
    test('threads comment created event under the issue post', {tag: '@jira_plugin'}, async ({jira}) => {
        // # Deliver an issue created webhook, then a comment created webhook for the same issue
        const created = await sendLegacyWebhook(await loadWebhookFixture('webhook-issue-created.json'), {team: jira.team.name, channel: jira.channel.name});
        expect(created.status).toBe(200);
        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id)).length).toBe(1);
        const [root] = await issuePosts(jira.userClient, jira.channel.id);
        const commented = await sendLegacyWebhook(await loadWebhookFixture('webhook-cloud-comment-created.json'), {
            team: jira.team.name,
            channel: jira.channel.name,
            params: {updated_comments: '1'},
        });
        expect(commented.status).toBe(200);

        // * Verify the comment is posted in the issue post's thread
        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id)).length).toBe(2);
        const [reply] = await issuePosts(jira.userClient, jira.channel.id);
        expect(reply.root_id).toBe(root.id);
        expect(headline(reply)).toContain('**commented** on story [TES-41: Unit test summary 1]');
        expect((reply.props.attachments as SlackAttachment[])[0].text).toContain('Added a comment');
    });

    /**
     * @objective Verify an issue resolved webhook, which changes resolution and status together, is posted by default with both changes.
     */
    test('posts issue resolved event with its status change', {tag: '@jira_plugin'}, async ({jira}) => {
        // # Deliver an issue resolved webhook
        const payload = await loadWebhookFixture('webhook-issue-updated-resolved.json');
        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name});
        expect(res.status).toBe(200);

        // * Verify one post lists the resolution and the status change
        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id)).length).toBe(1);
        const [post] = await issuePosts(jira.userClient, jira.channel.id);
        expect(headline(post)).toContain('**updated** story [TES-41: Unit test summary 1]');
        const [{fields}] = post.props.attachments as Array<{fields: Array<{value: string}>}>;
        expect(fields.map((f) => f.value)).toEqual(['**Resolved:** ~~Open~~ Done', '**Status:** ~~In Progress~~ Done']);
    });

    /**
     * @objective Verify an unresolved issue deleted webhook is posted by default.
     */
    test('posts issue deleted event to the channel', {tag: '@jira_plugin'}, async ({jira}) => {
        // # Deliver an issue deleted webhook
        const payload = await loadWebhookFixture('webhook-issue-deleted.json');
        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name});
        expect(res.status).toBe(200);

        // * Verify one post says the issue was deleted
        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id, 'IDT-19')).length).toBe(1);
        const [post] = await issuePosts(jira.userClient, jira.channel.id, 'IDT-19');
        expect(headline(post)).toContain('**deleted** task [IDT-19: hey]');
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
