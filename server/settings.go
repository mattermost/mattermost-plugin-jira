package main

import "github.com/mattermost/mattermost-server/model"

const (
	settingOn  = "on"
	settingOff = "off"
)

func (p *Plugin) settingsNotifications(ji Instance, mattermostUserId string, jiraUser JIRAUser, args []string) *model.CommandResponse {
	var value bool
	switch args[0] {
	case settingOn:
		value = true
	case settingOff:
		value = false
	default:
		return responsef("Invalid value. Accepted values are: `on` or `off`.")
	}

	if jiraUser.Settings == nil {
		jiraUser.Settings = &UserSettings{}
	}
	jiraUser.Settings.Notifications = value
	if err := p.StoreUserInfo(ji, mattermostUserId, jiraUser); err != nil {
		p.errorf("settingsNotifications, err: %v", err)
		responsef("Could not store new settings. Please contact your system administrator. error: %v", err)
	}

	// send back the actual value
	updatedJiraUser, err := p.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return responsef("Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}
	notifications := "off"
	if updatedJiraUser.Settings.Notifications {
		notifications = "on"
	}

	return responsef("Settings updated. Notifications %s.", notifications)
}
