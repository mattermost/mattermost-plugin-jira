// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

import InteractiveDialog from './base/interactive_dialog';

export default class JiraInteractiveDialog extends InteractiveDialog {
    constructor(page: Page) {
        super(page);
    }

    getLinkInHeader = async () => {
        const link = this.page.locator('#interactiveDialogModalIntroductionText a');
        return link.getAttribute('href');
    }
}
