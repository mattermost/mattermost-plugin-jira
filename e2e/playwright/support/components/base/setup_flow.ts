// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

import {ChannelsPage} from '@e2e-support/ui/pages';

import {DEFAULT_WAIT_MILLIS, screenshot} from '../../utils';

export default class SetupFlow {
    constructor(protected readonly page: Page, protected readonly channelsPage: ChannelsPage) {}

    clickFlowChoices = async (choices: string[]) => {
        for (const choice of choices) {
            await this.page.waitForTimeout(DEFAULT_WAIT_MILLIS);
            await screenshot(`post_action_before_${choice}`, this.page);
            await this.clickPostAction(choice);
            await screenshot(`post_action_after_${choice}`, this.page);
        }
    }

    clickPostAction = async (choice: string) => {
        const postElement = await this.channelsPage.getLastPost();
        await postElement.container.getByText(choice).last().click();
    }
}
