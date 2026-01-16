// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	settingOn  = "on"
	settingOff = "off"

	errStoreNewSettings = "Could not store new settings. Please contact your system administrator. Error: %v"
	errConnectToJira    = "Your account is not connected to Jira. Please type `/jira connect`. %v"

	assigneeRole = "assignee"
	mentionRole  = "mention"
	reporterRole = "reporter"
	watchingRole = "watching"
)

func (connection *Connection) updateRolesForDMNotification(role, roleStatus string) (string, bool) {
	switch role {
	case assigneeRole:
	case mentionRole:
	case reporterRole:
	case watchingRole:
	default:
		return fmt.Sprintf("* Invalid role `%s`. Accepted roles are: `assignee`, `mention`, `reporter` or `watching`.", role), false
	}

	var value bool
	switch roleStatus {
	case settingOn:
		value = true
	case settingOff:
		value = false
	default:
		return fmt.Sprintf("* Invalid value `%s`. Accepted values are: `on` or `off`.", roleStatus), false
	}

	if connection.Settings.RolesForDMNotification == nil {
		connection.Settings.RolesForDMNotification = make(map[string]bool)
	}
	connection.Settings.RolesForDMNotification[role] = value
	return "", true
}

func (p *Plugin) settingsNotifications(header *model.CommandArgs, instanceID, mattermostUserID types.ID, connection *Connection, args []string) *model.CommandResponse {
	helpTextPrefix := "`/jira settings notifications [assignee|mention|reporter|watching] [value]`\n" +
		"`/jira settings notifications fields [field1,field2,...]` - Set fields to notify on (empty to notify on all)\n"

	if len(args) < 2 {
		return p.response(header, helpTextPrefix+"* Invalid command args.")
	}

	if connection.Settings == nil {
		connection.Settings = &ConnectionSettings{}
	}

	// Handle "fields" subcommand
	if args[1] == "fields" {
		return p.settingsNotificationsFields(header, instanceID, mattermostUserID, connection, args)
	}

	// Handle role-based notifications
	if len(args) != 3 {
		return p.response(header, helpTextPrefix+"* Invalid command args.")
	}

	role, roleStatus := args[1], args[2]
	helpTextSuffix, isRoleUpdated := connection.updateRolesForDMNotification(role, roleStatus)
	if !isRoleUpdated {
		return p.response(header, helpTextPrefix+helpTextSuffix)
	}

	if err := p.userStore.StoreConnection(instanceID, mattermostUserID, connection); err != nil {
		p.errorf("settingsNotifications, err: %v", err)
		p.responsef(header, errStoreNewSettings, err)
	}

	// send back the actual value
	updatedConnection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		return p.responsef(header, errConnectToJira, err)
	}
	notifications := settingOff
	if updatedConnection.Settings.RolesForDMNotification[role] {
		notifications = settingOn
	}

	settingsUpdatedMsg := "Settings updated"

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		p.errorf("settingsNotifications, err: %v", err)
		p.responsef(header, errStoreNewSettings, err)
	}
	if len(instances.IDs()) > 1 {
		settingsUpdatedMsg += fmt.Sprintf(" for Jira instance %s", instanceID)
	}

	return p.responsef(header, "%s:\n* %s notifications %s.", settingsUpdatedMsg, cases.Title(language.Und, cases.NoLower).String(role), notifications)
}

func (p *Plugin) settingsNotificationsFields(header *model.CommandArgs, instanceID, mattermostUserID types.ID, connection *Connection, args []string) *model.CommandResponse {
	commonFields := "**Common fields:** `summary`, `description`, `priority`, `status`, `assignee`, `reporter`, `labels`, `components`, `fixversions`, `versions`, `resolution`, `duedate`, `Sprint`, `Story Points`\n\n" +
		"**Custom fields:** Use the field ID (e.g., `customfield_10001`) - check your Jira project settings for custom field IDs."

	if len(args) == 2 {
		fields := connection.Settings.FieldsForDMNotification
		if len(fields) == 0 {
			return p.responsef(header, "Field notifications: all fields (no filter set)\n\nUse `/jira settings notifications fields field1,field2,...` to filter.\nUse `/jira settings notifications fields list` to see available fields.\n\n%s", commonFields)
		}
		return p.responsef(header, "Field notifications enabled for: `%s`\n\nUse `/jira settings notifications fields clear` to receive notifications for all fields.\n\n%s", strings.Join(fields, ", "), commonFields)
	}

	fieldArg := args[2]

	if fieldArg == "list" {
		return p.responsef(header, "**Available fields for notifications:**\n\n%s", commonFields)
	}

	if fieldArg == "clear" {
		connection.Settings.FieldsForDMNotification = nil
		if err := p.userStore.StoreConnection(instanceID, mattermostUserID, connection); err != nil {
			p.errorf("settingsNotificationsFields, err: %v", err)
			return p.responsef(header, errStoreNewSettings, err)
		}
		return p.responsef(header, "Field filter cleared. You will now receive notifications for all field changes.")
	}

	// Parse comma-separated fields
	fields := strings.Split(fieldArg, ",")
	cleanFields := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			cleanFields = append(cleanFields, f)
		}
	}

	if len(cleanFields) == 0 {
		return p.responsef(header, "No valid fields provided. Use `/jira settings notifications fields field1,field2,...`")
	}

	connection.Settings.FieldsForDMNotification = cleanFields
	if err := p.userStore.StoreConnection(instanceID, mattermostUserID, connection); err != nil {
		p.errorf("settingsNotificationsFields, err: %v", err)
		return p.responsef(header, errStoreNewSettings, err)
	}

	return p.responsef(header, "Settings updated:\n* Field notifications enabled for: `%s`", strings.Join(cleanFields, ", "))
}
