package main

import "github.com/mattermost/mattermost-server/v5/model"

const (
	settingOn  = "on"
	settingOff = "off"
)

func (p *Plugin) settingsNotifications(header *model.CommandArgs, ji Instance, mattermostUserId string, jiraUser JIRAUser, args []string) *model.CommandResponse {
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

	if jiraUser.Settings == nil {
		jiraUser.Settings = &UserSettings{}
	}
	jiraUser.Settings.Notifications = value
	if err := p.userStore.StoreUserInfo(ji, mattermostUserId, jiraUser); err != nil {
		p.errorf("settingsNotifications, err: %v", err)
		p.responsef(header, "Could not store new settings. Please contact your system administrator. error: %v", err)
	}

	// send back the actual value
	updatedJiraUser, err := p.userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}
	notifications := "off"
	if updatedJiraUser.Settings.Notifications {
		notifications = "on"
	}

	return p.responsef(header, "Settings updated. Notifications %s.", notifications)
}
