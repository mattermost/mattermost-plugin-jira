// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

export default class InteractiveDialog {
    private readonly submitButton = this.page.locator('#interactiveDialogSubmit');

    constructor(protected readonly page: Page) {}

    fillTextField = async (field: string, value: string) => {
        await this.page.locator(`#${field}`).fill(value);
    }

    submit = async () => {
        await this.submitButton.click();
    }
}
