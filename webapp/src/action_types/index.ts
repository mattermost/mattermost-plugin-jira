// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from '../manifest';

const {id: PluginId} = manifest;

export default {
    OPEN_CONNECT_MODAL: `${PluginId}_open_connect_modal`,
    CLOSE_CONNECT_MODAL: `${PluginId}_close_connect_modal`,
    OPEN_DISCONNECT_MODAL: `${PluginId}_open_disconnect_modal`,
    CLOSE_DISCONNECT_MODAL: `${PluginId}_close_disconnect_modal`,

    CLOSE_CREATE_ISSUE_MODAL: `${PluginId}_close_create_modal`,
    OPEN_CREATE_ISSUE_MODAL: `${PluginId}_open_create_modal`,
    OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST: `${PluginId}_open_create_modal_without_post`,

    CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL: `${PluginId}_close_attach_modal`,
    OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL: `${PluginId}_open_attach_modal`,

    RECEIVED_CONNECTED: `${PluginId}_connected`,
    RECEIVED_INSTANCE_STATUS: `${PluginId}_instance_status`,
    RECEIVED_PLUGIN_SETTINGS: `${PluginId}_plugin_settings`,

    RECEIVED_JIRA_ISSUE_METADATA: `${PluginId}_received_metadata`,
    RECEIVED_JIRA_PROJECT_METADATA: `${PluginId}_received_projects`,
    CLEAR_JIRA_ISSUE_METADATA: `${PluginId}_clear_metadata`,

    OPEN_CHANNEL_SETTINGS: `${PluginId}_open_channel_settings`,
    CLOSE_CHANNEL_SETTINGS: `${PluginId}_close_channel_settings`,

    CREATED_CHANNEL_SUBSCRIPTION: `${PluginId}_created_channel_subscription`,
    EDITED_CHANNEL_SUBSCRIPTION: `${PluginId}_edited_channel_subscription`,

    CREATED_SUBSCRIPTION_TEMPLATE: `${PluginId}_created_subscription_template`,
    DELETED_SUBSCRIPTION_TEMPLATE: `${PluginId}_deleted_subscription_template`,
    EDITED_SUBSCRIPTION_TEMPLATE: `${PluginId}_edited_subscription_template`,
    RECEIVED_SUBSCRIPTION_TEMPLATES_PROJECT_KEY: `${PluginId}_received_subscription_templates_project_key`,

    RECEIVED_CHANNEL_SUBSCRIPTIONS: `${PluginId}_recevied_channel_subscriptions`,
    RECEIVED_SUBSCRIPTION_TEMPLATES: `${PluginId}_recevied_subscription_templates`,
    DELETED_CHANNEL_SUBSCRIPTION: `${PluginId}_deleted_channel_subscription`,
    SET_RHS_INSTANCE_ID: `${PluginId}_set_rhs_instance_id`,
    SET_RHS_TAB: `${PluginId}_set_rhs_tab`,
    SET_RHS_SORT: `${PluginId}_set_rhs_sort`,
    HYDRATE_RHS_VIEW_STATE: `${PluginId}_hydrate_rhs_view_state`,
    RHS_ISSUES_LOADING: `${PluginId}_rhs_issues_loading`,
    RECEIVED_RHS_ISSUES: `${PluginId}_received_rhs_issues`,
    RECEIVED_RHS_ISSUES_APPEND: `${PluginId}_received_rhs_issues_append`,
    RHS_ISSUES_ERROR: `${PluginId}_rhs_issues_error`,
};
