// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from '@playwright/test';

export default class JiraSiteAuthFlow {
    constructor(private readonly page: Page) {}

    login = async (email: string, password: string) => {
        await this.fillEmail(email);
        await this.submitEmail();
        await this.fillPassword(password);
        await this.submitPassword();
        await this.acceptPermissions();
    }

    private fillEmail = async (email: string) => {
        await this.page.getByPlaceholder('Enter your email').fill(email);
    }

    private submitEmail = async () => {
        await this.page.getByRole('button', {name: 'Continue', exact: true}).click();
    }

    private fillPassword = async (password: string) => {
        await this.page.getByPlaceholder('Enter password').fill(password);
    }

    private submitPassword = async () => {
        await this.page.getByRole('button', {name: 'Log in', exact: true}).click();
    }

    private acceptPermissions = async () => {
        await this.page.getByRole('button', {name: 'Accept', exact: true}).click();
    }
}
