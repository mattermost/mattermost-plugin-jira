package main

import "github.com/mattermost/mattermost-server/model"

const (
	settingOn  = "on"
	settingOff = "off"
)

func (p *Plugin) settingsNotifications(ji Instance, mattermostUserId string, jiraUser JIRAUser, args []string) *model.CommandResponse {
	const helpText = "`/jira settings notifications [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(args) != 2 {
		return responsef(helpText)
	}

	var value bool
	switch args[1] {
	case settingOn:
		value = true
	case settingOff:
		value = false
	default:
		return responsef(helpText)
	}

	if jiraUser.Settings == nil {
		jiraUser.Settings = &UserSettings{}
	}
	jiraUser.Settings.Notifications = value
	if err := p.userStore.StoreUserInfo(ji, mattermostUserId, jiraUser); err != nil {
		p.errorf("settingsNotifications, err: %v", err)
		responsef("Could not store new settings. Please contact your system administrator. error: %v", err)
	}

	// send back the actual value
	updatedJiraUser, err := p.userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return responsef("Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}
	notifications := "off"
	if updatedJiraUser.Settings.Notifications {
		notifications = "on"
	}

	return responsef("Settings updated. Notifications %s.", notifications)
}
