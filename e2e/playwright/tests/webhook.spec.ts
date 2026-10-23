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
    test('issue created event is posted to the channel', async ({pw, jira}) => {
        const payload = await loadWebhookFixture('webhook-issue-created.json');

        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name});
        expect(res.status).toBe(200);

        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id)).length).toBe(1);
        const [post] = await issuePosts(jira.userClient, jira.channel.id);
        const [attachment] = post.props.attachments as SlackAttachment[];
        expect(attachment.pretext).toContain('**created** story [TES-41: Unit test summary]');
        expect(attachment.text).toBe('Unit test description, not that long');

        const {page} = await pw.testBrowser.login(jira.user);
        await page.goto(`/${jira.team.name}/channels/${jira.channel.name}`);
        await expect(page.getByRole('link', {name: 'TES-41: Unit test summary'})).toBeVisible();
    });

    test('event types not enabled in the URL are skipped', async ({jira}) => {
        const payload = await loadWebhookFixture('webhook-issue-updated-renamed.json');

        const skipped = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name});
        expect(skipped.status).toBe(200);
        expect(await issuePosts(jira.userClient, jira.channel.id)).toHaveLength(0);

        const posted = await sendLegacyWebhook(payload, {
            team: jira.team.name,
            channel: jira.channel.name,
            params: {updated_summary: '1'},
        });
        expect(posted.status).toBe(200);
        await expect.poll(async () => (await issuePosts(jira.userClient, jira.channel.id)).length).toBe(1);
        const [post] = await issuePosts(jira.userClient, jira.channel.id);
        expect(post.message).toContain('**updated** summary from "Unit test summary" to "Unit test summary 1"');
    });

    test('wrong secret is rejected', async ({jira}) => {
        const payload = await loadWebhookFixture('webhook-issue-created.json');

        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: jira.channel.name, secret: 'nope'});

        expect(res.status).toBe(403);
        expect(await issuePosts(jira.userClient, jira.channel.id)).toHaveLength(0);
    });

    test('unknown channel is rejected', async ({jira}) => {
        const payload = await loadWebhookFixture('webhook-issue-created.json');

        const res = await sendLegacyWebhook(payload, {team: jira.team.name, channel: 'does-not-exist'});

        expect(res.status).toBe(400);
    });
});
