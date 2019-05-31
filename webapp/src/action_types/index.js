// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import PluginId from 'plugin_id';

export default {
    CLOSE_CREATE_ISSUE_MODAL: `${PluginId}_close_create_modal`,
    OPEN_CREATE_ISSUE_MODAL: `${PluginId}_open_create_modal`,

    CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL: `${PluginId}_close_attach_modal`,
    OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL: `${PluginId}_open_attach_modal`,

    RECEIVED_CONNECTED: `${PluginId}_connected`,
    RECEIVED_INSTANCE_STATUS: `${PluginId}_instance_status`,

    RECEIVED_JIRA_ISSUE_METADATA: `${PluginId}_received_metadata`,
    RECEIVED_JIRA_ISSUES: `${PluginId}_received_search_issues`,
};
