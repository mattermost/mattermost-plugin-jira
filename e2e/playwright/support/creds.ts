// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

const credsString = process.env.JIRA_OAUTH_CREDENTIALS;
if (!credsString) {
    console.error('Please provide base64-encoded credentials via env var JIRA_OAUTH_CREDENTIALS');
    process.exit(1);
}

type JiraCreds = {
    client_id: string;
    client_secret: string;
    email: string;
    password: string;
}

const creds: JiraCreds = JSON.parse(Buffer.from(credsString, 'base64').toString());

export const TEST_CLIENT_ID = creds.client_id;
export const TEST_CLIENT_SECRET = creds.client_secret;

export const JIRA_EMAIL = creds.email;
export const JIRA_PASSWORD = creds.password;
