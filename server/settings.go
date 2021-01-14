package main

import (
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	settingOn  = "on"
	settingOff = "off"
)

var onOffToBool = map[string]bool{settingOn: true, settingOff: false}

func (p *Plugin) settingsNotifications(header *model.CommandArgs, instanceID, mattermostUserID types.ID, connection *Connection, args []string) *model.CommandResponse {
	const helpText = "`/jira settings notifications [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 2 {
		return p.responsef(header, helpText)
	}

	value, ok := onOffToBool[args[1]]
	if !ok {
		return p.responsef(header, helpText)
	}

	if connection.Settings == nil {
		connection.Settings = &ConnectionSettings{}
	}
	connection.Settings.Notifications = value
	if err := p.userStore.StoreConnection(instanceID, mattermostUserID, connection); err != nil {
		p.errorf("settingsNotifications, err: %v", err)
		p.responsef(header, "Could not store new settings. Please contact your system administrator. error: %v", err)
	}

	// send back the actual value
	updatedConnection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}

	return p.responsef(header, "Settings updated. Notifications %s.",
		boolToOnOff(updatedConnection.Settings.Notifications))
}

func (p *Plugin) settingsSendNotificationsForAssigned(
	header *model.CommandArgs,
	instanceID, mattermostUserID types.ID,
	connection *Connection,
	args []string,
) *model.CommandResponse {
	const helpText = "`/jira settings send_notifications_for_assigned [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 2 {
		return p.responsef(header, helpText)
	}

	value, ok := onOffToBool[args[1]]
	if !ok {
		return p.responsef(header, helpText)
	}

	if connection.Settings == nil {
		connection.Settings = &ConnectionSettings{}
	}
	connection.Settings.SendNotificationsForAssigned = value
	if err := p.userStore.StoreConnection(instanceID, mattermostUserID, connection); err != nil {
		p.errorf("settingsSendNotificationsForAssigned, err: %v", err)
		p.responsef(header, "Could not store new settings. Please contact your system administrator. error: %v", err)
	}

	// send back the actual value
	updatedConnection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}

	return p.responsef(header, "Settings updated. SendNotificationsForAssigned %s.",
		boolToOnOff(updatedConnection.Settings.SendNotificationsForAssigned))
}

func (p *Plugin) settingsSendNotificationsForReporter(
	header *model.CommandArgs,
	instanceID, mattermostUserID types.ID,
	connection *Connection,
	args []string,
) *model.CommandResponse {
	const helpText = "`/jira settings send_notifications_for_reporter [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 2 {
		return p.responsef(header, helpText)
	}

	value, ok := onOffToBool[args[1]]
	if !ok {
		return p.responsef(header, helpText)
	}

	if connection.Settings == nil {
		connection.Settings = &ConnectionSettings{}
	}
	connection.Settings.SendNotificationsForReporter = value
	if err := p.userStore.StoreConnection(instanceID, mattermostUserID, connection); err != nil {
		p.errorf("settingsSendNotificationsForReporter, err: %v", err)
		p.responsef(header, "Could not store new settings. Please contact your system administrator. error: %v", err)
	}

	// send back the actual value
	updatedConnection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}

	return p.responsef(header, "Settings updated. SendNotificationsForReporter %s.",
		boolToOnOff(updatedConnection.Settings.SendNotificationsForReporter))
}

func boolToOnOff(isOn bool) string {
	if isOn {
		return settingOn
	}
	return settingOff
}
