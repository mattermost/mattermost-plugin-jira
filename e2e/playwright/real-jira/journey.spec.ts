// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {randomUUID} from 'node:crypto';

import {Client4} from '@mattermost/client';
import type {Channel} from '@mattermost/types/channels';
import type {Post} from '@mattermost/types/posts';
import type {Team} from '@mattermost/types/teams';
import type {UserProfile} from '@mattermost/types/users';
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

async function openChannel(pw: {testBrowser: {login: (u: UserProfile) => Promise<{page: Page}>}}) {
    const {page} = await pw.testBrowser.login({...user, password});
    await addAtlassianSession(page.context());
    await page.goto(`/${team.name}/channels/town-square`);
    return page;
}

async function runSlashCommand(page: Page, command: string) {
    const textbox = page.getByTestId('post_textbox');
    await textbox.fill(command);

    // Enter can be taken by the command or @mention autocomplete instead of sending.
    await expect(async () => {
        await textbox.press('Enter');
        await expect(textbox).toHaveValue('', {timeout: 2000});
    }).toPass({timeout: 15_000});
}

// The label's "for" doesn't reach react-select's generated input id, so find the input through its form group.
async function chooseOption(page: Page, form: Locator, field: string, search: string, option = search) {
    const input = form.locator('.form-group', {has: page.locator(`label[for="${field}"]`)}).locator('input:not([type="hidden"])').first();

    // Jira's search index can lag behind a just-created issue, so search again until it shows up.
    await expect(async () => {
        await input.fill('');
        await input.fill(search);
        await expect(page.getByRole('option', {name: option, exact: true})).toBeVisible({timeout: 5000});
    }).toPass({timeout: 60_000});
    await input.press('Enter');
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

test('user connects their Jira account through the real Atlassian consent screen', async ({pw}) => {
    const {page} = await pw.testBrowser.login({...user, password});
    await addAtlassianSession(page.context());

    await page.goto(`${mattermostURL}/plugins/${pluginId}/user/start`);
    await approveAtlassianConsent(page);

    await expect(page.getByText(/connected to Jira/i).first()).toBeVisible();
    await expect(page.getByText(jiraUser.displayName).first()).toBeVisible();
    expect(await pluginAPI('GET', '/api/v2/userinfo')).toMatchObject({is_connected: true});
});

test('user creates a Jira issue from Mattermost and it exists in Jira', async ({pw}) => {
    const summary = marker();
    const page = await openChannel(pw);
    await runSlashCommand(page, '/jira create');

    const modal = page.locator('.modal-content').filter({hasText: 'Create Jira Issue'});
    await expect(modal).toBeVisible();
    await chooseOption(page, modal, 'projects', project.name);
    await chooseOption(page, modal, 'issue_type', issueType.name);
    await modal.locator('#summary').fill(summary);
    await modal.locator('#submit-button').click();
    await expect(modal).toBeHidden();

    let key = '';
    await expect.poll(async () => {
        [key] = await jiraApi.searchKeys(realJira.admin, markerJQL(project.key, summary));
        return key;
    }, {message: 'issue should be searchable in Jira', timeout: 60_000}).toBeTruthy();
    createdIssueKeys.push(key);

    const issue = await jiraApi.getIssue(realJira.admin, key);
    expect(issue.fields.summary).toBe(summary);
    expect((issue.fields.reporter as {accountId: string}).accountId).toBe(jiraUser.accountId);
});

test('an issue created in Jira posts to a subscribed channel', async ({pw}) => {
    await subscribe(['event_created']);
    const {key, summary} = await createIssue();
    await deliverIssueCreated(key);

    const post = await waitForChannelPost(key);
    expect(JSON.stringify(post.props?.attachments)).toContain(`${key}: ${summary}`);

    const page = await openChannel(pw);
    await expect(page.getByText(`${key}: ${summary}`).first()).toBeVisible();
});

test('an issue that does not match the subscription filter is not posted', async () => {
    const unmatched = await createIssue(otherIssueType);
    const matched = await createIssue();
    await deliverIssueCreated(unmatched.key);
    await deliverIssueCreated(matched.key);

    // Several workers process webhooks, so give the earlier, unmatched one a moment after the matched one posts.
    await waitForChannelPost(matched.key);
    await new Promise((resolve) => setTimeout(resolve, 5000));
    expect(await channelPostsMentioning(unmatched.key)).toHaveLength(0);
});

test('hovering a Jira link shows the real issue in a tooltip', async ({pw}) => {
    const {key, summary} = await createIssue();
    const issueURL = `${realJira.siteURL}/browse/${key}`;
    await userClient.createPost({channel_id: channel.id, message: `Look at ${issueURL}`} as Post);

    const page = await openChannel(pw);
    await page.getByRole('link', {name: issueURL}).last().hover();

    await expect(page.locator('.jira-issue-tooltip .tooltip-ticket-summary')).toHaveText(summary);
});

test('user attaches a Mattermost message to a Jira issue as a comment', async ({pw}) => {
    const {key, summary} = await createIssue();
    const message = `${marker()} attached from Mattermost`;
    const post = await userClient.createPost({channel_id: channel.id, message} as Post);

    const page = await openChannel(pw);
    await page.locator(`#post_${post.id}`).hover();
    await page.locator(`#CENTER_actions_button_${post.id}`).click();
    await page.getByRole('menuitem', {name: 'Attach to Jira Issue'}).click();

    const modal = page.locator('.modal-content').filter({hasText: 'Attach Message to Jira Issue'});
    await expect(modal).toBeVisible();
    await chooseOption(page, modal, 'issue', summary, `${key}: ${summary}`);
    await modal.getByRole('button', {name: 'Attach'}).click();
    await expect(modal).toBeHidden();

    await expect.poll(async () => (await jiraApi.commentBodies(realJira.user, key)).some((body) => body.includes(message)), {
        message: `a comment on ${key} with the Mattermost message`,
        timeout: 60_000,
    }).toBe(true);
});

test('a comment added in Jira posts to a subscribed channel', async ({pw}) => {
    await subscribe(['event_created_comment']);
    const {key} = await createIssue();
    const body = `${marker()} commented in Jira`;
    const comment = await jiraApi.addComment(realJira.user, key, body);
    const issue = await jiraApi.getIssue(realJira.user, key);
    await deliverWebhook({webhookEvent: 'comment_created', comment, issue});

    const post = await waitForChannelPost(body);
    expect(JSON.stringify(post.props?.attachments)).toContain('commented');

    const page = await openChannel(pw);
    await expect(page.getByText(body).first()).toBeVisible();
});

test('user transitions and assigns a Jira issue with slash commands', async ({pw}) => {
    const {key} = await createIssue();
    const page = await openChannel(pw);

    await runSlashCommand(page, `/jira transition ${key} done`);
    await expect(page.getByText(/transitioned to/).last()).toBeVisible();
    await expect.poll(async () => ((await jiraApi.getIssue(realJira.user, key)).fields.status as {name: string}).name, {timeout: 30_000}).toBe('Done');

    await runSlashCommand(page, `/jira assign ${key} @${user.username}`);
    await expect(page.getByText(/assigned to Jira issue/).last()).toBeVisible();
    await expect.poll(async () => ((await jiraApi.getIssue(realJira.user, key)).fields.assignee as {accountId?: string} | null)?.accountId, {timeout: 30_000}).toBe(jiraUser.accountId);
});

test('an expired access token is refreshed with Atlassian and the new one is saved', async () => {
    const before = readConnection(realJira.siteURL, user.id);
    const token = before?.OAuth2Token;
    expect(token, 'stored OAuth2 token').toBeTruthy();
    writeConnection(realJira.siteURL, user.id, {...before, OAuth2Token: {...token!, expiry: new Date(Date.now() - 60_000).toISOString()}});

    await pluginAPI('GET', `/api/v2/get-jira-project-metadata?instance_id=${encodeURIComponent(realJira.siteURL)}`);

    const after = readConnection(realJira.siteURL, user.id)?.OAuth2Token;
    expect(after?.access_token).not.toBe(token!.access_token);
    expect(after?.refresh_token).not.toBe(token!.refresh_token);
    expect(new Date(after!.expiry).getTime()).toBeGreaterThan(Date.now());
    expect(await pluginAPI('GET', '/api/v2/userinfo')).toMatchObject({is_connected: true});
});

test('user disconnects their Jira account', async ({pw}) => {
    const page = await openChannel(pw);
    await runSlashCommand(page, '/jira disconnect');

    await expect(page.getByText('You have successfully disconnected your Jira account').last()).toBeVisible();
    expect(await pluginAPI('GET', '/api/v2/userinfo')).toMatchObject({is_connected: false});
    expect(readConnection(realJira.siteURL, user.id)).toBeUndefined();
});
