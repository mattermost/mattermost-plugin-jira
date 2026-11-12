// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {baseGlobalSetup} from '@mattermost/playwright-lib';

import {startFakeJira} from './fake-jira/server';
import {loginAdmin} from './support/admin';
import {fakeCloudSite, fakeJiraControlURL, fakeJiraPort, fakeJiraURL} from './support/constants';
import {fakeJira} from './support/fake_jira';
import {installPlugin} from './support/plugin';

export default async function globalSetup() {
    await baseGlobalSetup();

    const teardown = await startOrReuseFakeJira();

    const adminClient = await loginAdmin();
    await installPlugin(adminClient, 'e2e');

    return teardown;
}

// A fake already running on the port (the Cursor cloud agent's `.cursor/fake-jira.ts`) is reset and left running.
async function startOrReuseFakeJira(): Promise<() => Promise<void>> {
    const running = await fetch(`${fakeJiraControlURL}/status`).then((res) => res.ok, () => false);
    if (running) {
        await fakeJira.reset();
        return async () => {};
    }

    const fake = await startFakeJira({port: fakeJiraPort, baseURL: fakeJiraURL, cloud: fakeCloudSite});
    return fake.close;
}
