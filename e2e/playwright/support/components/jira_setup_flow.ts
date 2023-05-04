// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

import {ChannelsPage} from '@e2e-support/ui/pages';

import {getSlackAttachmentLocatorId} from '../utils';

import SetupFlow from './base/setup_flow';

export default class JiraSetupFlow extends SetupFlow {
    constructor(page: Page, channelsPage: ChannelsPage) {
        super(page, channelsPage);
    }

    clickConnectLink = async () => {
        const post = await this.channelsPage.getLastPost();
        const postId = await post.getId();
        const locatorId = getSlackAttachmentLocatorId(postId);

        const connectLink = this.page.locator(`${locatorId} a`).getByText('here');
        await connectLink.click();
    }
}
