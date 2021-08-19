package main

import (
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	settingOn  = "on"
	settingOff = "off"

	errStoreNewSettings = "Could not store new settings. Please contact your system administrator. error: %v"
	errConnectToJira    = "Your username is not connected to Jira. Please type `/jira connect`. %v"
)

func (p *Plugin) settingsNotifications(header *model.CommandArgs, instanceID, mattermostUserID types.ID, connection *Connection, args []string) *model.CommandResponse {
	const helpText = "`/jira settings notifications [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 2 {
		return p.responsef(header, helpText)
	}

	var value bool
	switch args[1] {
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
	connection.Settings.Notifications = value
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
	if updatedConnection.Settings.Notifications {
		notifications = settingOn
	}

	return p.responsef(header, "Settings updated. Notifications %s.", notifications)
}

func (p *Plugin) settingsWatching(header *model.CommandArgs, instanceID, mattermostUserID types.ID, connection *Connection, args []string) *model.CommandResponse {
	const helpText = "`/jira watching [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 2 {
		return p.responsef(header, helpText)
	}

	var value bool
	switch args[1] {
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
	connection.Settings.Watching = &value
	if err := p.userStore.StoreConnection(instanceID, mattermostUserID, connection); err != nil {
		p.errorf("settingsWatching, err: %v", err)
		p.responsef(header, errStoreNewSettings, err)
	}

	return p.responsef(header, "Settings updated. Watching %s.", value)
}
