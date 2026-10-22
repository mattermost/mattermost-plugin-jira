// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/* eslint-disable no-console */

import fs from 'node:fs';
import path from 'node:path';

import {chromium, type Locator, type Page} from '@playwright/test';

import {realJira} from './env';
import {generateTotp} from './totp';

const sessionMaxAgeMs = 3 * 24 * 60 * 60 * 1000;

export const canSignInAutomatically = () => Boolean(realJira.user.password && realJira.user.totpSecret);

// Walks Atlassian's login and Microsoft SSO as the regular test user. Without a password/TOTP secret it only
// fills the email and waits for a person to finish in the browser.
export async function signInToAtlassian(page: Page, timeoutMs: number) {
    const siteHost = new URL(realJira.siteURL).hostname;
    await page.goto(`https://id.atlassian.com/login?continue=${encodeURIComponent(`${realJira.siteURL}/jira`)}`);
    await completeSignIn(page, timeoutMs, async () => {
        const url = new URL(page.url());
        return url.hostname === siteHost && !url.pathname.includes('login');
    });
}

// Answers whichever Atlassian/Microsoft sign-in screen is showing until isDone() holds.
export async function completeSignIn(page: Page, timeoutMs: number, isDone: () => Promise<boolean>) {
    const {email, password, totpSecret} = realJira.user;
    let lastCode = '';

    // Microsoft keeps every step's inputs in the page and hides the inactive ones off screen.
    const active = (selector: string) => page.locator(`${selector}:not(.moveOffScreen):not([aria-hidden="true"])`).first();

    const steps: Array<[Locator, (el: Locator) => Promise<void>]> = [
        [page.locator('input[name="username"]').first(), async (el) => {
            await el.fill(email);
            await page.keyboard.press('Enter');
        }],
        [active('input[name="loginfmt"]'), async (el) => {
            await el.fill(email);
            await page.locator('#idSIButton9').click();
        }],
        [active('input[name="passwd"]'), async (el) => {
            if (password) {
                await el.fill(password);
                await page.locator('#idSIButton9').click();
            }
        }],
        [active('[data-value="PhoneAppOTP"]'), async (el) => {
            if (totpSecret) {
                await el.click();
            }
        }],
        [active('input[name="otc"]'), async (el) => {
            if (!totpSecret) {
                return;
            }

            // Microsoft rejects a code that was already used, so wait for the next 30s window.
            let code = generateTotp(totpSecret);
            while (code === lastCode) {
                await page.waitForTimeout(2000);
                code = generateTotp(totpSecret);
            }
            lastCode = code;
            await el.fill(code);
            await page.locator('#idSubmit_SAOTCC_Continue').click();
        }],
        [active('#KmsiDescription, #kmsiTitle'), async () => {
            await page.locator('#idSIButton9').click();
        }],
    ];

    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
        if (await isDone()) {
            return;
        }
        for (const [el, act] of steps) {
            if (await el.isVisible().catch(() => false)) {
                console.log(`Sign-in step: ${el}`);
                await act(el);
                await page.waitForLoadState('domcontentloaded').catch(() => undefined);
                break;
            }
        }
        await page.waitForTimeout(1000);
    }
    throw new Error(`Sign-in did not finish; last page was ${page.url()}`);
}

export async function saveSession(page: Page) {
    fs.mkdirSync(path.dirname(realJira.userSessionPath), {recursive: true});
    await page.context().storageState({path: realJira.userSessionPath});
}

// Refreshes the saved session when it's missing or old and the suite can sign in by itself.
export async function ensureAtlassianSession() {
    const age = fs.existsSync(realJira.userSessionPath) ? Date.now() - fs.statSync(realJira.userSessionPath).mtimeMs : Infinity;
    if (age < sessionMaxAgeMs) {
        return;
    }
    if (!canSignInAutomatically()) {
        if (age === Infinity) {
            throw new Error('No saved Atlassian session. Run `npm run real:login`, or set JIRA_CLOUD_USER_PASSWORD and JIRA_CLOUD_USER_TOTP_SECRET.');
        }
        console.warn('The saved Atlassian session is over 3 days old; if connect fails, run `npm run real:login`.');
        return;
    }

    const browser = await chromium.launch();
    try {
        const page = await browser.newPage();
        await signInToAtlassian(page, 2 * 60_000);
        await saveSession(page);
        console.log('Signed in to Atlassian automatically and saved the session');
    } finally {
        await browser.close();
    }
}
