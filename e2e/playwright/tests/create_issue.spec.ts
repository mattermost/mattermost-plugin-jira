// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {fakeIssues, fakeProject} from '../fake-jira/state';
import {jiraFlavor} from '../support/constants';
import {fakeJira} from '../support/fake_jira';
import {expect, test} from '../support/fixture';
import {connectUserViaKV} from '../support/jira';
import {openChannel, openPostAction, pickOption} from '../support/ui';

const postMessage = 'Checkout button overlaps the cart total on a 375px viewport';
const summary = 'Checkout button hides the cart total';
const createdKey = `${fakeProject.key}-${fakeIssues.length + 1}`;

test.describe('create issue', () => {
    test.skip(jiraFlavor === 'server', 'fake-jira only serves the Cloud project APIs behind the modals');

    test.beforeEach(async ({jira}) => {
        await connectUserViaKV(jira.user.id);
    });

    /**
     * @objective Verify an issue created from a post's message actions is sent to Jira and linked in the post's thread.
     */
    test('creates issue from the post menu', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Open Create Jira Issue on a post and fill in project, issue type and summary
        const {page, post} = await openPostAction(pw, jira, postMessage, 'Create Jira issue');
        const dialog = page.getByRole('dialog', {name: 'Create Jira issue'});
        await pickOption(page, dialog.getByRole('combobox', {name: 'Project'}), fakeProject.name);
        await pickOption(page, dialog.getByRole('combobox', {name: 'Issue Type'}), 'Task');
        await dialog.getByRole('textbox', {name: 'Summary'}).fill(summary);
        await dialog.getByRole('button', {name: 'Create'}).click();

        // * Verify the modal closes
        await expect(dialog).toBeHidden();

        // * Verify Jira received the issue with the chosen fields and a link back to the post
        const [created] = await fakeJira.requestsTo('POST', '/rest/api/2/issue');
        expect(created.body).toMatchObject({fields: {project: {key: fakeProject.key}, issuetype: {id: '10001'}, summary}});
        const {description} = (created.body as {fields: {description: string}}).fields;
        expect(description).toContain(postMessage);
        expect(description).toContain(`/${jira.team.name}/pl/${post.id}`);

        // * Verify the new issue is linked in a reply to the post
        await expect.poll(async () => {
            const thread = await jira.userClient.getPostThread(post.id);
            return Object.values(thread.posts).map((p) => p.message);
        }).toContainEqual(expect.stringContaining(`Created a Jira issue: [${createdKey}: ${summary}`));
    });

    /**
     * @objective Verify the create issue modal blocks submission without a summary and sends nothing to Jira.
     */
    test('shows required field error when summary is missing', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Choose project and issue type, then submit without a summary
        const {page} = await openPostAction(pw, jira, postMessage, 'Create Jira issue');
        const dialog = page.getByRole('dialog', {name: 'Create Jira issue'});
        await pickOption(page, dialog.getByRole('combobox', {name: 'Project'}), fakeProject.name);
        await pickOption(page, dialog.getByRole('combobox', {name: 'Issue Type'}), 'Task');
        await dialog.getByRole('button', {name: 'Create'}).click();

        // * Verify the error is shown and the modal stays open
        await expect(dialog.getByText('This field is required.')).toBeVisible();
        await expect(dialog).toBeVisible();

        // * Verify no issue was sent to Jira
        expect(await fakeJira.requestsTo('POST', '/rest/api/2/issue')).toEqual([]);
    });

    /**
     * @objective Verify /jira create opens the create issue modal with the command text as description and creates the issue in the channel.
     */
    test('opens create issue modal from /jira create', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Run /jira create with a description
        const {page, channelsPage} = await openChannel(pw, jira);
        await channelsPage.postMessage('/jira create Nightly export stalls on archived channels');
        const dialog = page.getByRole('dialog', {name: 'Create Jira issue'});

        // # Choose project and issue type
        await pickOption(page, dialog.getByRole('combobox', {name: 'Project'}), fakeProject.name);
        await pickOption(page, dialog.getByRole('combobox', {name: 'Issue Type'}), 'Bug');

        // * Verify the description is prefilled from the command
        await expect(dialog.getByRole('textbox', {name: 'Description'})).toHaveValue('Nightly export stalls on archived channels');

        // # Fill in the summary and create the issue
        await dialog.getByRole('textbox', {name: 'Summary'}).fill('Nightly export stalls');
        await dialog.getByRole('button', {name: 'Create'}).click();
        await expect(dialog).toBeHidden();

        // * Verify Jira received the description unchanged, since no post is attached
        const [created] = await fakeJira.requestsTo('POST', '/rest/api/2/issue');
        expect(created.body).toMatchObject({fields: {issuetype: {id: '10002'}, summary: 'Nightly export stalls', description: 'Nightly export stalls on archived channels'}});

        // * Verify the issue link is posted to the channel by the user, outside any thread
        await expect.poll(async () => {
            const {posts} = await jira.userClient.getPosts(jira.channel.id);
            return Object.values(posts).filter((p) => p.user_id === jira.user.id && !p.root_id).map((p) => p.message);
        }).toContainEqual(expect.stringContaining(`Created a Jira issue: [${createdKey}: Nightly export stalls (To Do)]`));
    });

    /**
     * @objective Verify the create issue modal shows Jira's failure and stays open when Jira rejects the create request.
     */
    test('shows error in modal when Jira fails to create the issue', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Make Jira fail issue creation, then submit a complete issue
        await fakeJira.fault('POST', '/rest/api/2/issue', 500);
        const {page} = await openPostAction(pw, jira, postMessage, 'Create Jira issue');
        const dialog = page.getByRole('dialog', {name: 'Create Jira issue'});
        await pickOption(page, dialog.getByRole('combobox', {name: 'Project'}), fakeProject.name);
        await pickOption(page, dialog.getByRole('combobox', {name: 'Issue Type'}), 'Task');
        await dialog.getByRole('textbox', {name: 'Summary'}).fill(summary);
        await dialog.getByRole('button', {name: 'Create'}).click();

        // * Verify the error is shown in the modal
        await expect(dialog.getByText(/failed to create issue/)).toBeVisible();
        await expect(dialog.getByRole('button', {name: 'Create'})).toBeEnabled();

        // * Verify no issue link was posted
        const {posts} = await jira.userClient.getPosts(jira.channel.id);
        expect(Object.values(posts).map((p) => p.message)).not.toContainEqual(expect.stringContaining('Created a Jira issue'));
    });

    /**
     * @objective Verify a post attached to an existing issue is sent to Jira as a comment and linked in the post's thread.
     */
    test('attaches message to an existing issue', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Open Attach to Jira Issue on a post and pick an issue
        const {page, post} = await openPostAction(pw, jira, postMessage, 'Attach to Jira Issue');
        const dialog = page.getByRole('dialog', {name: 'Attach message to Jira issue'});
        await dialog.getByRole('combobox', {name: 'Jira Issue'}).click();
        await page.getByRole('option', {name: new RegExp(fakeIssues[0].key)}).click();
        await dialog.getByRole('button', {name: 'Attach'}).click();

        // * Verify the modal closes
        await expect(dialog).toBeHidden();

        // * Verify Jira received a comment on the issue with the post's text and a link back to it
        const [comment] = await fakeJira.requestsTo('POST', `/rest/api/2/issue/${fakeIssues[0].key}/comment`);
        const {body} = comment.body as {body: string};
        expect(body).toContain(postMessage);
        expect(body).toContain(`/${jira.team.name}/pl/${post.id}`);

        // * Verify the post's thread links the issue
        await expect.poll(async () => {
            const thread = await jira.userClient.getPostThread(post.id);
            return Object.values(thread.posts).map((p) => p.message);
        }).toContainEqual(expect.stringContaining(`Message attached to [${fakeIssues[0].key}]`));
    });
});

/**
 * @objective Verify /jira create tells a user without a Jira connection to connect instead of opening the modal.
 */
test('asks unconnected user to connect on /jira create', {tag: '@jira_plugin'}, async ({pw, jira}) => {
    // # Run /jira create without a Jira connection
    const {page, channelsPage} = await openChannel(pw, jira);
    await channelsPage.postMessage('/jira create Nightly export stalls');

    // * Verify the user is asked to connect and no modal opens
    await expect((await channelsPage.centerView.getLastPost()).body).toContainText('Your Mattermost account is not connected to Jira.');
    await expect(page.getByRole('dialog', {name: 'Create Jira issue'})).toBeHidden();
});
