package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	settingOn  = "on"
	settingOff = "off"

	errStoreNewSettings = "Could not store new settings. Please contact your system administrator. error: %v"
	errConnectToJira    = "Your username is not connected to Jira. Please type `/jira connect`. %v"
	subCommandAssignee  = "assignee"
	subCommandMention   = "mention"
	subCommandReporter  = "reporter"
	subCommandWatching  = "watching"
)

func (p *Plugin) settingsNotifications(header *model.CommandArgs, instanceID, mattermostUserID types.ID, connection *Connection, args []string) *model.CommandResponse {
	const helpText = "`/jira settings notifications [assignee|mention|reporter|watching] [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 3 {
		return p.responsef(header, helpText)
	}

	var value bool
	switch args[2] {
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
	switch args[1] {
	case subCommandAssignee:
		connection.Settings.SendNotificationsForAssignee = &value
	case subCommandMention:
		connection.Settings.SendNotificationsForMention = &value
	case subCommandReporter:
		connection.Settings.SendNotificationsForReporter = &value
	case subCommandWatching:
		connection.Settings.SendNotificationsForWatching = &value
	default:
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
	switch args[1] {
	case subCommandAssignee:
		if *updatedConnection.Settings.SendNotificationsForAssignee {
			notifications = settingOn
		}
	case subCommandMention:
		if *updatedConnection.Settings.SendNotificationsForMention {
			notifications = settingOn
		}
	case subCommandReporter:
		if *updatedConnection.Settings.SendNotificationsForReporter {
			notifications = settingOn
		}
	case subCommandWatching:
		if *updatedConnection.Settings.SendNotificationsForWatching {
			notifications = settingOn
		}
	}

	return p.responsef(header, "Settings updated.\n\t%s notifications %s.", strings.Title(args[1]), notifications)
}
