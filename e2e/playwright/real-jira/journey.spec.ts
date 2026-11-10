// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {randomUUID} from 'node:crypto';

import {Client4} from '@mattermost/client';
import type {Channel} from '@mattermost/types/channels';
import type {Post} from '@mattermost/types/posts';
import type {Team} from '@mattermost/types/teams';
import type {UserProfile} from '@mattermost/types/users';
import type {ChannelsPage, PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Locator, Page} from '@playwright/test';

import {loginAdmin} from '../support/admin';
import {mattermostURL, pluginId, webhookSecret} from '../support/constants';
import {expect, test} from '../support/fixture';
import {instancePathSegment} from '../support/jira';
import {readConnection, writeConnection} from '../support/kv';

import {addAtlassianSession, approveAtlassianConsent, jiraApi, markerJQL} from './atlassian';
import {realJira} from './env';

// One user's journey against real Jira; each step builds on the previous one.
test.describe.configure({mode: 'serial'});

const password = 'passwd';
const createdIssueKeys: string[] = [];

let team: Team;
let channel: Channel;
let user: UserProfile;
let userClient: Client4;
let jiraUser: {accountId: string; displayName: string};
let project: {key: string; name: string};
let issueType: {id: string; name: string};
let otherIssueType: {id: string; name: string};

const marker = () => `${realJira.issueMarker} ${randomUUID().slice(0, 8)}`;

test.beforeAll(async () => {
    const adminClient = await loginAdmin();
    const id = randomUUID().slice(0, 8);
    team = await adminClient.createTeam({name: `jira-real-${id}`, display_name: `Jira real ${id}`, type: 'O'} as Team);
    user = await adminClient.createUser({username: `jira-real-${id}`, email: `jira-real-${id}@example.com`, password} as UserProfile, '', '');
    await adminClient.addToTeam(team.id, user.id);
    channel = await adminClient.getChannelByName(team.id, 'town-square');

    userClient = new Client4();
    userClient.setUrl(mattermostURL);
    await userClient.login(user.username, password);

    jiraUser = await jiraApi.myself(realJira.user);
    project = await jiraApi.project(realJira.admin, realJira.projectKey);
    const types = await jiraApi.issueTypes(realJira.admin, realJira.projectKey);
    issueType = types.find((t) => t.name === 'Task') ?? types[0];
    otherIssueType = types.find((t) => t.name === 'Bug') ?? types.find((t) => t.id !== issueType.id)!;
});

test.afterAll(async () => {
    for (const key of createdIssueKeys) {
        await jiraApi.deleteIssue(realJira.admin, key).catch(() => undefined);
    }
});

async function openChannel(pw: PlaywrightExtended) {
    const {page, channelsPage} = await pw.testBrowser.login({...user, password});
    await addAtlassianSession(page.context());
    await channelsPage.goto(team.name, 'town-square');
    return {page, channelsPage};
}

async function runSlashCommand(channelsPage: ChannelsPage, command: string) {
    const textbox = channelsPage.centerView.postCreate.input;
    await textbox.fill(command);

    // Enter can be taken by the command or @mention autocomplete instead of sending.
    await expect(async () => {
        await textbox.press('Enter');
        await expect(textbox).toHaveValue('', {timeout: 2000});
    }).toPass({timeout: 15_000});
}

async function chooseOption(page: Page, form: Locator, field: string, search: string, option = search) {
    const input = form.getByRole('combobox', {name: field});

    // Jira's search index can lag behind a just-created issue, so search again until it shows up.
    await expect(async () => {
        await input.fill('');
        await input.fill(search);
        await expect(page.getByRole('option', {name: option, exact: true})).toBeVisible({timeout: 5000});
    }).toPass({timeout: 60_000});
    await page.getByRole('option', {name: option, exact: true}).click();
}

async function pluginAPI(method: string, path: string, body?: unknown) {
    const res = await fetch(`${mattermostURL}/plugins/${pluginId}${path}`, {
        method,
        headers: {Authorization: `Bearer ${userClient.getToken()}`, 'Content-Type': 'application/json'},
        body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!res.ok) {
        throw new Error(`plugin ${method} ${path} failed: ${res.status} ${await res.text()}`);
    }
    return res.json();
}

function subscribe(events: string[], issueTypeIds = [issueType.id]) {
    return pluginAPI('POST', '/api/v2/subscriptions/channel', {
        name: `${realJira.issueMarker} ${randomUUID().slice(0, 8)}`,
        channel_id: channel.id,
        instance_id: realJira.siteURL,
        filters: {events, projects: [project.key], issue_types: issueTypeIds, fields: []},
    });
}

async function createIssue(type = issueType) {
    const summary = marker();
    const {key} = await jiraApi.createIssue(realJira.user, {projectKey: project.key, issueTypeId: type.id, summary});
    createdIssueKeys.push(key);
    return {key, summary};
}

// Jira can't reach a local Mattermost, so tests deliver the webhook Jira would send. For created issues and
// comments the plugin re-fetches the issue from Jira before posting, so a post only appears if that real call succeeds.
async function deliverWebhook(payload: Record<string, unknown>) {
    const res = await fetch(`${mattermostURL}/plugins/${pluginId}/instance/${instancePathSegment(realJira.siteURL)}/api/v2/webhook?secret=${webhookSecret}`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({timestamp: Date.now(), ...payload}),
    });
    expect(res.status).toBe(200);
}

async function deliverIssueCreated(key: string) {
    const issue = await jiraApi.getIssue(realJira.user, key);
    await deliverWebhook({webhookEvent: 'jira:issue_created', issue_event_type_name: 'issue_created', user: issue.fields.creator, issue});
}

async function channelPostsMentioning(text: string): Promise<Post[]> {
    const {order, posts} = await userClient.getPosts(channel.id);
    return order.map((id) => posts[id]).filter((p) => JSON.stringify([p.message, p.props?.attachments]).includes(text));
}

async function waitForChannelPost(text: string) {
    await expect.poll(async () => (await channelPostsMentioning(text)).length, {message: `a post mentioning ${text}`, timeout: 60_000}).toBeGreaterThan(0);
    return (await channelPostsMentioning(text))[0];
}

/**
 * @objective Verify a user connects their Jira account through the real Atlassian consent screen.
 *
 * @precondition
 * The real test site, its OAuth app and both test accounts are configured in .env.
 */
test('connects Jira account through the real Atlassian consent screen', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Start the connect flow with the regular user's Atlassian session and approve the consent screen
    const {page} = await pw.testBrowser.login({...user, password});
    await addAtlassianSession(page.context());
    await page.goto(`${mattermostURL}/plugins/${pluginId}/user/start`);
    await approveAtlassianConsent(page);

    // * Verify the Jira account is connected
    await expect(page.getByText(/connected to Jira/i).first()).toBeVisible();
    await expect(page.getByText(jiraUser.displayName).first()).toBeVisible();
    expect(await pluginAPI('GET', '/api/v2/userinfo')).toMatchObject({is_connected: true});
});

/**
 * @objective Verify an issue created from the Mattermost modal exists in Jira with the right reporter.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('creates Jira issue from Mattermost and finds it in Jira', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Open the create issue modal and submit an issue
    const summary = marker();
    const {page, channelsPage} = await openChannel(pw);
    await runSlashCommand(channelsPage, '/jira create');
    const modal = page.getByRole('dialog', {name: 'Create Jira Issue'});
    await expect(modal).toBeVisible();
    await chooseOption(page, modal, 'Project', project.name);
    await chooseOption(page, modal, 'Issue Type', issueType.name);
    await modal.getByRole('textbox', {name: 'Summary'}).fill(summary);
    await modal.getByRole('button', {name: 'Create'}).click();
    await expect(modal).toBeHidden();

    // * Verify the issue is searchable in Jira
    let key = '';
    await expect.poll(async () => {
        [key] = await jiraApi.searchKeys(realJira.admin, markerJQL(project.key, summary));
        return key;
    }, {message: 'issue should be searchable in Jira', timeout: 60_000}).toBeTruthy();
    createdIssueKeys.push(key);

    // * Verify its summary and reporter
    const issue = await jiraApi.getIssue(realJira.admin, key);
    expect(issue.fields.summary).toBe(summary);
    expect((issue.fields.reporter as {accountId: string}).accountId).toBe(jiraUser.accountId);
});

/**
 * @objective Verify an issue created in Jira is posted to a channel subscribed to issue creation.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('posts issue created in Jira to a subscribed channel', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Subscribe to issue creation, create an issue in Jira and deliver its webhook
    await subscribe(['event_created']);
    const {key, summary} = await createIssue();
    await deliverIssueCreated(key);

    // * Verify the channel gets a post for the issue
    const post = await waitForChannelPost(key);
    expect(JSON.stringify(post.props?.attachments)).toContain(`${key}: ${summary}`);

    // * Verify the post is visible in the channel
    const {page} = await openChannel(pw);
    await expect(page.getByText(`${key}: ${summary}`).first()).toBeVisible();
});

/**
 * @objective Verify an issue of a type outside the subscription filter is not posted.
 *
 * @precondition
 * The channel is subscribed to issue creation for one issue type.
 */
test('skips issue that does not match the subscription filter', {tag: '@jira_plugin'}, async () => {
    // # Create one issue of another type and one matching issue, then deliver both webhooks
    const unmatched = await createIssue(otherIssueType);
    const matched = await createIssue();
    await deliverIssueCreated(unmatched.key);
    await deliverIssueCreated(matched.key);

    // * Verify only the matching issue is posted
    // Several workers process webhooks, so give the earlier, unmatched one a moment after the matched one posts.
    await waitForChannelPost(matched.key);
    await new Promise((resolve) => setTimeout(resolve, 5000));
    expect(await channelPostsMentioning(unmatched.key)).toHaveLength(0);
});

/**
 * @objective Verify hovering a Jira issue link shows the real issue in the plugin's tooltip.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('shows real issue in tooltip when hovering a Jira link', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Post a link to a new issue and hover it
    const {key, summary} = await createIssue();
    const issueURL = `${realJira.siteURL}/browse/${key}`;
    await userClient.createPost({channel_id: channel.id, message: `Look at ${issueURL}`} as Post);
    const {page} = await openChannel(pw);
    await page.getByRole('link', {name: issueURL}).last().hover();

    // * Verify the tooltip shows the issue summary
    await expect(page.getByRole('heading', {name: summary})).toBeVisible();
});

/**
 * @objective Verify a Mattermost message attached to a Jira issue becomes a comment on it.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('attaches Mattermost message to a Jira issue as a comment', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Open Attach to Jira Issue from a post's message actions and attach it to a new issue
    const {key, summary} = await createIssue();
    const message = `${marker()} attached from Mattermost`;
    const created = await userClient.createPost({channel_id: channel.id, message} as Post);
    const {page, channelsPage} = await openChannel(pw);
    const post = await channelsPage.centerView.getPostById(created.id);
    await post.hover();
    await post.postMenu.actionsButton.click();
    await page.getByRole('menuitem', {name: 'Attach to Jira Issue'}).click();
    const modal = page.getByRole('dialog', {name: 'Attach Message to Jira Issue'});
    await expect(modal).toBeVisible();
    await chooseOption(page, modal, 'Jira Issue', summary, `${key}: ${summary}`);
    await modal.getByRole('button', {name: 'Attach'}).click();
    await expect(modal).toBeHidden();

    // * Verify the issue has a comment with the message
    await expect.poll(async () => (await jiraApi.commentBodies(realJira.user, key)).some((body) => body.includes(message)), {
        message: `a comment on ${key} with the Mattermost message`,
        timeout: 60_000,
    }).toBe(true);
});

/**
 * @objective Verify a comment added in Jira is posted to a channel subscribed to comments.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('posts comment added in Jira to a subscribed channel', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Subscribe to comments, comment on a new issue in Jira and deliver the webhook
    await subscribe(['event_created_comment']);
    const {key} = await createIssue();
    const body = `${marker()} commented in Jira`;
    const comment = await jiraApi.addComment(realJira.user, key, body);
    const issue = await jiraApi.getIssue(realJira.user, key);
    await deliverWebhook({webhookEvent: 'comment_created', comment, issue});

    // * Verify the channel gets a post for the comment
    const post = await waitForChannelPost(body);
    expect(JSON.stringify(post.props?.attachments)).toContain('commented');

    // * Verify the post is visible in the channel
    const {page} = await openChannel(pw);
    await expect(page.getByText(body).first()).toBeVisible();
});

/**
 * @objective Verify slash commands transition and assign a real Jira issue.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('transitions and assigns Jira issue with slash commands', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Transition a new issue to Done
    const {key} = await createIssue();
    const {page, channelsPage} = await openChannel(pw);
    await runSlashCommand(channelsPage, `/jira transition ${key} done`);

    // * Verify the confirmation and the status in Jira
    await expect(page.getByText(/transitioned to/).last()).toBeVisible();
    await expect.poll(async () => ((await jiraApi.getIssue(realJira.user, key)).fields.status as {name: string}).name, {timeout: 30_000}).toBe('Done');

    // # Assign the issue to the user
    await runSlashCommand(channelsPage, `/jira assign ${key} @${user.username}`);

    // * Verify the confirmation and the assignee in Jira
    await expect(page.getByText(/assigned to Jira issue/).last()).toBeVisible();
    await expect.poll(async () => ((await jiraApi.getIssue(realJira.user, key)).fields.assignee as {accountId?: string} | null)?.accountId, {timeout: 30_000}).toBe(jiraUser.accountId);
});

/**
 * @objective Verify an expired access token is refreshed with Atlassian and the new one is stored.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('refreshes expired access token with Atlassian and saves the new one', {tag: '@jira_plugin'}, async () => {
    // # Expire the stored token and make a call that needs Jira
    const before = readConnection(realJira.siteURL, user.id);
    const token = before?.OAuth2Token;
    expect(token, 'stored OAuth2 token').toBeTruthy();
    writeConnection(realJira.siteURL, user.id, {...before, OAuth2Token: {...token!, expiry: new Date(Date.now() - 60_000).toISOString()}});
    await pluginAPI('GET', `/api/v2/get-jira-project-metadata?instance_id=${encodeURIComponent(realJira.siteURL)}`);

    // * Verify a new token pair was saved and the user is still connected
    const after = readConnection(realJira.siteURL, user.id)?.OAuth2Token;
    expect(after?.access_token).not.toBe(token!.access_token);
    expect(after?.refresh_token).not.toBe(token!.refresh_token);
    expect(new Date(after!.expiry).getTime()).toBeGreaterThan(Date.now());
    expect(await pluginAPI('GET', '/api/v2/userinfo')).toMatchObject({is_connected: true});
});

/**
 * @objective Verify /jira disconnect removes the user's Jira connection.
 *
 * @precondition
 * The user is connected to the real test site.
 */
test('disconnects Jira account with a slash command', {tag: '@jira_plugin'}, async ({pw}) => {
    // # Run /jira disconnect
    const {page, channelsPage} = await openChannel(pw);
    await runSlashCommand(channelsPage, '/jira disconnect');

    // * Verify the confirmation and that the connection is gone
    await expect(page.getByText('You have successfully disconnected your Jira account').last()).toBeVisible();
    expect(await pluginAPI('GET', '/api/v2/userinfo')).toMatchObject({is_connected: false});
    expect(readConnection(realJira.siteURL, user.id)).toBeUndefined();
});
