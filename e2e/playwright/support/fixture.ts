// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Client4} from '@mattermost/client';
import type {Channel} from '@mattermost/types/channels';
import type {Team} from '@mattermost/types/teams';
import type {UserProfile} from '@mattermost/types/users';
import {expect, test as base} from '@mattermost/playwright-lib';

import {fakeJira} from './fake_jira';

export {expect};

type JiraFixture = {
    adminClient: Client4;
    user: UserProfile;
    userClient: Client4;
    team: Team;
    channel: Channel;
};

export type FailureScreenshotOptions = {failureScreenshots: boolean};

// pw.initSetup() is deliberately avoided: it resets server config, which drops the plugin's settings and PluginStates.
export const test = base.extend<{jira: JiraFixture; failureScreenshot: void}, FailureScreenshotOptions>({
    failureScreenshots: [false, {option: true, scope: 'worker'}],

    // Playwright's own failure screenshot runs after pw closes the logged-in context, so it only ever sees a blank
    // page. Depending on pw tears this down first, while the test's pages are still open.
    failureScreenshot: [async ({pw, failureScreenshots}, use, testInfo) => {
        await use();
        if (!failureScreenshots || testInfo.status === testInfo.expectedStatus) {
            return;
        }
        for (const [i, page] of (pw.testBrowser.context?.pages() ?? []).entries()) {
            const body = await page.screenshot({fullPage: true}).catch(() => undefined);
            if (body) {
                await testInfo.attach(`failure-page-${i}`, {body, contentType: 'image/png'});
            }
        }
    }, {auto: true}],

    jira: async ({pw}, use) => {
        await fakeJira.reset();

        const {adminClient} = await pw.getAdminClient();
        const team = await pw.createNewTeam(adminClient, {name: 'jira', displayName: 'Jira', type: 'O', unique: true});
        const user = await pw.createNewUserProfile(adminClient, {prefix: 'jira', disableTutorial: true, disableOnboarding: true});
        await adminClient.addToTeam(team.id, user.id);
        const {client: userClient} = await pw.makeClient(user);
        if (!userClient) {
            throw new Error(`Failed to log in ${user.username}`);
        }
        const channel = await adminClient.getChannelByName(team.id, 'town-square');

        await use({adminClient, user, userClient, team, channel});

        expect(await fakeJira.contractViolations(), 'fake-jira Cloud traffic must match Atlassian\'s OpenAPI spec').toEqual([]);
    },
});
