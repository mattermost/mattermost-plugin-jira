// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

import {ChannelsPage} from '@e2e-support/ui/pages';

import SetupFlow from './base/setup_flow';

export default class JiraSetupFlow extends SetupFlow {
    constructor(page: Page, channelsPage: ChannelsPage) {
        super(page, channelsPage);
    }

    clickConnectLink = async () => {
        const link = this.page.getByRole('link', { name: 'here' });
        return link.click();
    }
}
