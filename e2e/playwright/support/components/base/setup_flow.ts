// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

import {ChannelsPage} from '@e2e-support/ui/pages';

import {DEFAULT_WAIT_MILLIS} from '../../utils';

export default class SetupFlow {
    constructor(protected readonly page: Page, protected readonly channelsPage: ChannelsPage) {}

    clickFlowChoices = async (choices: string[]) => {
        for (const choice of choices) {
            await this.page.waitForTimeout(DEFAULT_WAIT_MILLIS);
            await this.clickPostAction(choice);
        }
    }

    clickPostAction = async (choice: string) => {
        const postElement = await this.channelsPage.getLastPost();
        await postElement.container.getByText(choice).last().click();
    }
}
