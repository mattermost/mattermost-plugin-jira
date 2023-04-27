// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

export const JIRA_SITE_URL = process.env.MM_JIRA_PLUGIN_E2E_JIRA_URL!;
if (!JIRA_SITE_URL) {
    console.error('Please provide a Jira site URL via env var MM_JIRA_PLUGIN_E2E_JIRA_URL');
    process.exit(1);
}

export const JIRA_CLIENT_ID = process.env.MM_JIRA_PLUGIN_E2E_CLIENT_ID!;
if (!JIRA_CLIENT_ID) {
    console.error('Please provide a Jira OAuth app client id via env var MM_JIRA_PLUGIN_E2E_CLIENT_ID');
    process.exit(1);
}

export const JIRA_CLIENT_SECRET = process.env.MM_JIRA_PLUGIN_E2E_CLIENT_SECRET!;
if (!JIRA_CLIENT_SECRET) {
    console.error('Please provide a Jira OAuth app client secret via env var MM_JIRA_PLUGIN_E2E_CLIENT_SECRET');
    process.exit(1);
}

export const JIRA_EMAIL = process.env.MM_JIRA_PLUGIN_E2E_EMAIL!;
if (!JIRA_EMAIL) {
    console.error('Please provide a Jira user email via env var MM_JIRA_PLUGIN_E2E_EMAIL');
    process.exit(1);
}

export const JIRA_PASSWORD = process.env.MM_JIRA_PLUGIN_E2E_PASSWORD!;
if (!JIRA_PASSWORD) {
    console.error('Please provide a Jira user password via env var MM_JIRA_PLUGIN_E2E_PASSWORD');
    process.exit(1);
}
