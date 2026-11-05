// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {fakeCloudUser, fakeIssues} from '../fake-jira/state';
import {jiraFlavor, jiraInstanceURL, mattermostURL, pluginId} from '../support/constants';
import {fakeJira} from '../support/fake_jira';
import {expect, test} from '../support/fixture';
import {connectUserViaKV} from '../support/jira';
import {readConnection} from '../support/kv';
import {runSlashCommand} from '../support/ui';

const issue = fakeIssues[0];

test.describe('issue commands', () => {
    test.skip(jiraFlavor === 'server', 'fake-jira only serves Cloud users and issues');

    test.beforeEach(async ({jira}) => {
        await connectUserViaKV(jira.user.id);
    });

    /**
     * @objective Verify /jira transition moves an issue to the matching state in Jira.
     */
    test('transitions issue with /jira transition', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Transition the issue by a partial state name
        const {reply} = await runSlashCommand(pw, jira, `/jira transition ${issue.key} progress`);

        // * Verify the reply names the new state
        await expect(reply).toContainText(`${issue.key} transitioned to In Progress`);

        // * Verify Jira received the transition to In Progress
        const [transition] = await fakeJira.requestsTo('POST', `/rest/api/2/issue/${issue.key}/transitions`);
        expect(transition.body).toMatchObject({transition: {id: '3'}});
    });

    /**
     * @objective Verify /jira assign looks up the Jira user and assigns them to the issue.
     */
    test('assigns issue with /jira assign', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Assign the issue by a search string
        const {reply} = await runSlashCommand(pw, jira, `/jira assign ${issue.key} cloud`);

        // * Verify the reply names the assignee
        await expect(reply).toContainText(`${fakeCloudUser.displayName} assigned to Jira issue ${issue.key}`);

        // * Verify Jira was searched and received the assignee's account ID
        const [search] = await fakeJira.requestsTo('GET', '/rest/api/2/user/assignable/search');
        expect(search.query).toMatchObject({issueKey: issue.key, query: 'cloud'});
        const [assign] = await fakeJira.requestsTo('PUT', `/rest/api/2/issue/${issue.key}/assignee`);
        expect(assign.body).toMatchObject({accountId: fakeCloudUser.accountId});
    });

    /**
     * @objective Verify /jira unassign clears the issue's assignee in Jira.
     */
    test('unassigns issue with /jira unassign', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Unassign the issue
        const {reply} = await runSlashCommand(pw, jira, `/jira unassign ${issue.key}`);

        // * Verify the reply confirms it
        await expect(reply).toContainText(`Unassigned Jira issue ${issue.key}`);

        // * Verify Jira received an assignee update without an account
        const [unassign] = await fakeJira.requestsTo('PUT', `/rest/api/2/issue/${issue.key}/assignee`);
        expect(unassign.body).not.toHaveProperty('accountId');
    });

    /**
     * @objective Verify /jira view shows the issue as a card with its actions.
     */
    test('shows issue card with /jira view', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # View the issue
        const {reply} = await runSlashCommand(pw, jira, `/jira view ${issue.key}`);

        // * Verify the card shows the issue and its actions
        await expect(reply.getByRole('link', {name: `${issue.key}: ${issue.summary}`})).toBeVisible();
        await expect(reply).toContainText(issue.description.replaceAll('*', ''));
        await expect(reply.getByRole('button', {name: 'Share publicly'})).toBeVisible();
    });
});

test.describe('account commands', () => {
    /**
     * @objective Verify /jira disconnect removes the user's Jira connection.
     */
    test('disconnects user with /jira disconnect', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Connect, then disconnect
        await connectUserViaKV(jira.user.id);
        const {reply} = await runSlashCommand(pw, jira, '/jira disconnect');

        // * Verify the reply confirms the disconnection
        await expect(reply).toContainText('You have successfully disconnected your Jira account');

        // * Verify the plugin reports the user as disconnected
        const info = await fetch(`${mattermostURL}/plugins/${pluginId}/api/v2/userinfo`, {
            headers: {Authorization: `Bearer ${jira.userClient.getToken()}`},
        });
        expect(await info.json()).toMatchObject({is_connected: false});
    });

    /**
     * @objective Verify /jira settings notifications turns off a notification role and stores it on the connection.
     */
    test('turns off assignee notifications with /jira settings', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Connect, then turn off assignee notifications
        await connectUserViaKV(jira.user.id);
        const {reply} = await runSlashCommand(pw, jira, '/jira settings notifications assignee off');

        // * Verify the reply confirms the new value
        await expect(reply).toContainText('Assignee notifications off.');

        // * Verify the setting is stored on the connection
        expect(readConnection(jiraInstanceURL, jira.user.id)?.Settings).toMatchObject({RolesForDMNotification: {assignee: false}});
    });

    /**
     * @objective Verify /jira help and /jira instance list only show admin commands and installed instances to system admins.
     */
    test('shows admin help and instance list to system admins only', {tag: '@jira_plugin'}, async ({pw, jira}) => {
        // # Ask for help as a regular user
        const {reply, channelsPage} = await runSlashCommand(pw, jira, '/jira help');

        // * Verify the user commands are listed without the admin section
        await expect(reply).toContainText('Mattermost Jira Plugin - Slash Command Help');
        await expect(reply).toContainText('/jira [issue] view [issue-key]');
        await expect(reply).not.toContainText('For System Administrators');

        // # List instances as a regular user
        await channelsPage.postMessage('/jira instance list');

        // * Verify the command is refused
        await expect(reply).toContainText('/jira instance list can only be run by a system administrator.');

        // # Promote the user, then ask for help and list instances again
        await jira.adminClient.updateUserRoles(jira.user.id, 'system_user system_admin');
        await channelsPage.postMessage('/jira help');

        // * Verify the admin section is listed
        await expect(reply).toContainText('For System Administrators');

        // # List instances again
        await channelsPage.postMessage('/jira instance list');

        // * Verify the installed instance is listed
        await expect(reply.getByRole('table')).toContainText(jiraInstanceURL);
    });
});
