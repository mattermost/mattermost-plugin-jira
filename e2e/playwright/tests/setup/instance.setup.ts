// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {testConfig} from '@mattermost/playwright-lib';

import {jiraFlavor} from '../../support/constants';
import {test as setup} from '../../support/fixture';
import {installCloudInstance, installServerInstance, uninstallInstances} from '../../support/jira';

setup(`install ${jiraFlavor} Jira instance`, async ({pw}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    if (!adminUser) {
        throw new Error('Admin user is missing; global setup should have created it');
    }
    const [team] = await adminClient.getMyTeams();
    const channel = await adminClient.getChannelByName(team.id, 'town-square');

    await uninstallInstances(adminClient, channel.id, team.id);

    if (jiraFlavor === 'server') {
        await installServerInstance(adminClient, channel.id, team.id);
        return;
    }

    const {page} = await pw.testBrowser.login({...adminUser, password: testConfig.adminPassword});
    await installCloudInstance(adminClient, page, team.name, channel.id, team.id);
});
