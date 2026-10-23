// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {baseGlobalSetup} from '@mattermost/playwright-lib';

import {startFakeJira} from './fake-jira/server';
import {loginAdmin} from './support/admin';
import {fakeCloudSite, fakeJiraPort, fakeJiraURL} from './support/constants';
import {installPlugin} from './support/plugin';

export default async function globalSetup() {
    await baseGlobalSetup();

    const fake = await startFakeJira({port: fakeJiraPort, baseURL: fakeJiraURL, cloud: fakeCloudSite});

    const adminClient = await loginAdmin();
    await installPlugin(adminClient);

    return fake.close;
}
