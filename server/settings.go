package main

import (
	"fmt"

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

func (connection *Connection) updateRolesForDMNotification(role string, hasNotification bool) bool {
	if role != assigneeRole && role != mentionRole && role != reporterRole && role != watchingRole {
		return false
	}
	if connection.Settings.RolesForDMNotification == nil {
		connection.Settings.RolesForDMNotification = make(map[string]bool)
	}
	connection.Settings.RolesForDMNotification[role] = hasNotification
	return true
}

func (p *Plugin) settingsNotifications(header *model.CommandArgs, instanceID, mattermostUserID types.ID, connection *Connection, args []string) *model.CommandResponse {
	const helpText = "`/jira settings notifications [assignee|mention|reporter|watching] [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 3 {
		return p.responsef(header, helpText)
	}

	role, roleStatus := args[1], args[2]
	var value bool
	switch roleStatus {
	case settingOn:
		value = true
	case settingOff:
		value = false
	default:
		return p.responsef(header, helpText)
	}

	if connection.Settings == nil {
		connection.Settings = &ConnectionSettings{}
	}
	if !connection.updateRolesForDMNotification(role, value) {
		return p.responsef(header, helpText)
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

	return p.responsef(header, "%s.\n\t%s notifications %s.", settingsUpdatedMsg, cases.Title(language.Und, cases.NoLower).String(role), notifications)
}
