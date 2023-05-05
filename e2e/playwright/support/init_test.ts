// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import path from 'node:path';
import fs from 'node:fs';

import {test} from '@e2e-support/test_fixture';
import {cleanUpBotDMs} from './utils';

import {clearKVStoreForPlugin} from './kv';
import {DeepPartial} from '@mattermost/types/utilities';
import {AdminConfig} from '@mattermost/types/config';

const pluginDistPath = path.join(__dirname, '../../../dist');
const pluginId = 'jira';

// # Clear plugin's KV store
test.beforeAll(async () => {
    if (process.env.AVOID_TEST_CLEANUP === 'true') {
        return;
    }

    await clearKVStoreForPlugin(pluginId);
});

// # Upload plugin
test.beforeEach(async ({pw}) => {
    const files = await fs.promises.readdir(pluginDistPath);
    const bundle = files.find((fname) => fname.endsWith('.tar.gz'));
    if (!bundle) {
        throw new Error('Failed to find plugin bundle in dist folder');
    }

    const bundlePath = path.join(pluginDistPath, bundle);
    const {adminClient} = await pw.getAdminClient();

    await adminClient.uploadPluginX(bundlePath, true);
    await adminClient.enablePlugin(pluginId);
});

// # Clear bot DM channel
test.beforeEach(async ({pw}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    await cleanUpBotDMs(adminClient, adminUser!.id, pluginId);
});

type JiraPluginSettings = {
    DisplaySubscriptionNameInNotifications: boolean;
    EnableAutocomplete: boolean;
    EnableWebhookEventLogging: boolean;
    GroupsAllowedToEditJiraSubscriptions: string;
    HideDecriptionComment: boolean;
    JiraAdminAdditionalHelpText: string;
    MaxAttachmentSize: string;
    RolesAllowedToEditJiraSubscriptions: string;
    displaysubscriptionnameinnotifications: boolean;
    enableautocomplete: boolean;
    enablejiraui: boolean;
    groupsallowedtoeditjirasubscriptions: string;
    hidedecriptioncomment: boolean;
    jiraadminadditionalhelptext: string;
    rolesallowedtoeditjirasubscriptions: string;
    secret: string;
};

const pluginConfig: JiraPluginSettings = {
    DisplaySubscriptionNameInNotifications: false,
    EnableAutocomplete: true,
    EnableWebhookEventLogging: false,
    GroupsAllowedToEditJiraSubscriptions: '',
    HideDecriptionComment: false,
    JiraAdminAdditionalHelpText: '',
    MaxAttachmentSize: '',
    RolesAllowedToEditJiraSubscriptions: 'system_admin',
    displaysubscriptionnameinnotifications: false,
    enableautocomplete: true,
    enablejiraui: true,
    groupsallowedtoeditjirasubscriptions: '',
    hidedecriptioncomment: false,
    jiraadminadditionalhelptext: '',
    rolesallowedtoeditjirasubscriptions: 'system_admin',
    secret: '',
};

// # Set plugin settings
test.beforeAll(async ({pw}) => {
    if (process.env.AVOID_TEST_CLEANUP === 'true') {
        return
    }

    const {adminClient} = await pw.getAdminClient();

    const config = await adminClient.getConfig();
    const newConfig: DeepPartial<AdminConfig> = {
        PluginSettings: {
            ...config.PluginSettings,
            Plugins: {
                ...config.PluginSettings.Plugins,
                [pluginId]: pluginConfig as any,
            },
        },
    };

    await adminClient.patchConfig(newConfig);
    await adminClient.disablePlugin(pluginId);
    await adminClient.enablePlugin(pluginId);
});

// # Log in
test.beforeEach(async ({pw}) => {
    const {adminClient, adminUser} = await pw.getAdminClient();
    if (!adminUser) {
        throw new Error('Failed to get admin user');
    }

    await adminClient.patchConfig({
        ServiceSettings: {
            EnableTutorial: false,
            EnableOnboardingFlow: false,
        },
    });
});
