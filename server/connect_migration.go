// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	adminNoticeLegacyConnect   = "legacy_connect_"
	prefixConnectUsersMigrated = "connect_users_migrated_"
)

const legacyConnectInstanceNotice = ":warning: **Action needed: reinstall Jira Cloud %[1]s**\n\n" +
	"This Jira instance was installed with the Atlassian Connect app. Atlassian ends support for Connect on January 31, 2027, " +
	"and the Jira plugin no longer supports it. Until you reinstall, users can't connect to this instance or use Jira commands with it.\n\n" +
	"To fix it:\n" +
	"1. Run `/jira setup`, choose **Jira Cloud (OAuth 2.0)** and enter %[1]s as the Jira URL.\n" +
	"2. Remove the old Mattermost app from Jira in [Manage apps](%[2]s).\n\n" +
	"Channel subscriptions are kept. When the reinstall is done, users who connected before get a message asking them to run `/jira connect`."

// migrateAwayFromConnect runs on activation. It tells admins which instances
// still need an OAuth 2.0 reinstall and what is needed for notifications, and
// asks users of already reinstalled instances to reconnect.
func (p *Plugin) migrateAwayFromConnect(instances *Instances) {
	for _, id := range instances.IDs() {
		instance, err := p.instanceStore.LoadInstance(id)
		if err != nil {
			p.client.Log.Warn("Failed to load Jira instance for the Connect migration", "instance", id.String(), "error", err.Error())
			continue
		}

		switch instance.(type) {
		case *cloudInstance:
			p.notifyAdminsOnce(adminNoticeLegacyConnect, instance.GetID(),
				fmt.Sprintf(legacyConnectInstanceNotice, instance.GetJiraBaseURL(), instance.GetManageAppsURL()))
		case *cloudOAuthInstance:
			if p.markOnce(prefixConnectUsersMigrated, instance.GetID()) {
				if err := p.disconnectConnectUsers(instance.GetID()); err != nil {
					p.unmarkOnce(prefixConnectUsersMigrated, instance.GetID())
				}
			}
		}

		if instance.Common().IsCloudInstance() {
			p.notifyAdminsIfAPITokenMissing(instance)
		}
	}
}

// disconnectConnectUsers disconnects users of an OAuth 2.0 instance whose
// connection was made with the Atlassian Connect app, and asks them to reconnect.
// Users found before a listing error are still processed.
func (p *Plugin) disconnectConnectUsers(instanceID types.ID) error {
	var userIDs []types.ID
	err := p.userStore.MapUsers(func(user *User) error {
		if user.ConnectedInstances == nil || !user.ConnectedInstances.checkIfExists(instanceID) {
			return nil
		}
		connection, err := p.userStore.LoadConnection(instanceID, user.MattermostUserID)
		if err != nil || connection.OAuth2Token != nil {
			return nil
		}
		userIDs = append(userIDs, user.MattermostUserID)
		return nil
	})
	if err != nil {
		p.client.Log.Warn("Failed to list users for the Connect migration", "instance", instanceID.String(), "error", err.Error())
	}

	for _, userID := range userIDs {
		p.disconnectUserWithNotice(userID, instanceID, connectConnectionRemovedNotice)
	}
	return err
}
