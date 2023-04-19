// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

export default class JiraSiteAuthFlow {
    constructor(private readonly page: Page) {}

    fillEmail = async (email: string) => {
        await this.page.getByPlaceholder('Enter your email').fill(email);
    }

    submitEmail = async () => {
        await this.page.getByRole('button', {name: 'Continue', exact: true}).click();
    }

    fillPassword = async (password: string) => {
        await this.page.getByPlaceholder('Enter password').fill(password);
    }

    submitPassword = async () => {
        await this.page.getByRole('button', {name: 'Log in', exact: true}).click();
    }

    acceptPermissions = async () => {
        await this.page.getByRole('button', {name: 'Accept', exact: true}).click();
    }
}
