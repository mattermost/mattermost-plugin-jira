// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execFileSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import path from 'node:path';

import {pluginId} from './constants';

export type InstanceType = 'cloud-oauth' | 'server';

export type JiraAccount = {
    accountId?: string;
    name?: string;
    key?: string;
    displayName: string;
    emailAddress?: string;
    active?: boolean;
};

export type SeededCredentials =
    | {oauth2: {access_token: string; refresh_token: string; expires_in: number}}
    | {oauth1: {token: string; secret: string}};

type SeedConnectionOptions = {
    mattermostUserId: string;
    instanceId: string;
    instanceType: InstanceType;
    jiraAccount: JiraAccount;
    credentials: SeededCredentials;
};

const composeDir = path.join(__dirname, '..');

// Writes the records connectUser() would store (server/user.go), so tests start with a connected user
// without driving the OAuth flow. Must stay in sync with server/kv.go key derivation.
export function seedConnection({mattermostUserId, instanceId, instanceType, jiraAccount, credentials}: SeedConnectionOptions) {
    const connection = {
        ...jiraAccount,
        ...('oauth2' in credentials ? {OAuth2Token: {
            access_token: credentials.oauth2.access_token,
            token_type: 'Bearer',
            refresh_token: credentials.oauth2.refresh_token,
            expiry: new Date(Date.now() + (credentials.oauth2.expires_in * 1000)).toISOString(),
        }} : {
            Oauth1AccessToken: credentials.oauth1.token,
            Oauth1AccessSecret: credentials.oauth1.secret,
        }),
        Settings: {notifications: true, RolesForDMNotification: null},
        mattermost_user_id: mattermostUserId,
    };
    const user = {
        mattermost_user_id: mattermostUserId,
        connected_instances: [{InstanceID: instanceId, Alias: '', Type: instanceType, IsV2Legacy: false, SetupWizardUserID: ''}],
    };
    const jiraAccountId = jiraAccount.accountId || jiraAccount.name;
    if (!jiraAccountId) {
        throw new Error('jiraAccount needs an accountId (Cloud) or name (Data Center)');
    }

    upsert({
        [md5(`${instanceId}/${mattermostUserId}`)]: connection,
        [md5(`${instanceId}/${jiraAccountId}`)]: mattermostUserId,
        [`user_${md5(mattermostUserId)}`]: user,
    });
}

export type StoredConnection = Record<string, unknown> & {
    OAuth2Token?: {access_token: string; refresh_token: string; expiry: string};
};

export function readConnection(instanceId: string, mattermostUserId: string): StoredConnection | undefined {
    const key = md5(`${instanceId}/${mattermostUserId}`);
    const out = psql(`SELECT convert_from(pvalue, 'UTF8') FROM pluginkeyvaluestore WHERE pluginid = '${pluginId}' AND pkey = ${quote(key)};`, ['-A', '-t']);
    return out.trim() ? JSON.parse(out) : undefined;
}

export function writeConnection(instanceId: string, mattermostUserId: string, connection: StoredConnection) {
    upsert({[md5(`${instanceId}/${mattermostUserId}`)]: connection});
}

function upsert(entries: Record<string, unknown>) {
    const values = Object.entries(entries).map(([key, value]) => `('${pluginId}', ${quote(key)}, convert_to(${quote(JSON.stringify(value))}, 'UTF8'), 0)`);
    psql(`INSERT INTO pluginkeyvaluestore (pluginid, pkey, pvalue, expireat) VALUES ${values.join(', ')}
        ON CONFLICT (pluginid, pkey) DO UPDATE SET pvalue = EXCLUDED.pvalue, expireat = 0;`);
}

function psql(sql: string, flags: string[] = []) {
    return execFileSync('docker', ['compose', 'exec', '-T', 'postgres', 'psql', '-U', 'mmuser', '-d', 'mattermost_test', '-v', 'ON_ERROR_STOP=1', '-q', ...flags], {
        cwd: composeDir,
        input: sql,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'inherit'],
    });
}

function quote(value: string) {
    return `'${value.replaceAll("'", "''")}'`;
}

function md5(value: string) {
    return createHash('md5').update(value).digest('hex');
}
