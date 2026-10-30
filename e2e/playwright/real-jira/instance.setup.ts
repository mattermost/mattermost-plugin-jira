// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {testConfig} from '@mattermost/playwright-lib';

import {mattermostURL, pluginId} from '../support/constants';
import {test as setup} from '../support/fixture';
import {installCloudInstance, instancePathSegment, uninstallInstances} from '../support/jira';

import {realJira} from './env';

setup('admin installs the real Jira Cloud instance through the setup wizard', async ({pw}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    if (!adminUser) {
        throw new Error('Admin user is missing; global setup should have created it');
    }
    const [team] = await adminClient.getMyTeams();
    const channel = await adminClient.getChannelByName(team.id, 'town-square');

    await uninstallInstances(adminClient, channel.id, team.id);

    const {page} = await pw.testBrowser.login({...adminUser, password: testConfig.adminPassword});
    await installCloudInstance(adminClient, page, team.name, channel.id, team.id, {siteURL: realJira.siteURL, ...realJira.oauth});

    const res = await fetch(`${mattermostURL}/plugins/${pluginId}/instance/${instancePathSegment(realJira.siteURL)}/user/connect`, {
        headers: {Authorization: `Bearer ${adminClient.getToken()}`},
        redirect: 'manual',
    });
    const location = res.headers.get('location') ?? '';
    if (!location.startsWith('https://auth.atlassian.com/')) {
        throw new Error(`Connect redirects to "${location}" instead of auth.atlassian.com. dist/ holds the fake-Jira E2E build; run \`make dist\` first.`);
    }
});
