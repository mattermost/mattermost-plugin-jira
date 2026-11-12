// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Runs the E2E suite's fake Jira on its own so an agent can use the plugin from a browser.
// The Playwright suite reuses it when it is already running (and resets its state).
//
//   npx --yes tsx@4.23.15 ../../.cursor/fake-jira.ts           start on :8090 (run from e2e/playwright)
//   npx --yes tsx@4.23.15 ../../.cursor/fake-jira.ts --reset   reset a running fake to its seeded state

/* eslint-disable no-console */

import {startFakeJira} from '../e2e/playwright/fake-jira/server';
import {fakeCloudSite, fakeJiraControlURL, fakeJiraPort, fakeJiraURL} from '../e2e/playwright/support/constants';
import {fakeJira} from '../e2e/playwright/support/fake_jira';

async function main() {
    if (process.argv.includes('--reset')) {
        await fakeJira.reset();
        console.log(`[fake-jira] Reset the fake at ${fakeJiraControlURL}.`);
        return;
    }

    const fake = await startFakeJira({port: fakeJiraPort, baseURL: fakeJiraURL, cloud: fakeCloudSite});
    console.log([
        '[fake-jira] Ready.',
        `  Jira Server URL:     ${fakeJiraURL}`,
        `  Jira Cloud site URL: ${fakeCloudSite.siteURL}`,
        `  Cloud OAuth client:  ${fakeCloudSite.clientId} / ${fakeCloudSite.clientSecret}`,
        `  Control API:         ${fakeJiraControlURL}/__control/`,
    ].join('\n'));

    const stop = async () => {
        await fake.close();
        process.exit(0);
    };
    process.on('SIGINT', stop);
    process.on('SIGTERM', stop);
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});
