package main

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	settingOn  = "on"
	settingOff = "off"
)

func (p *Plugin) settingsNotifications(a *Action) (string, error) {
	const helpText = "`/jira settings notifications [value]`\n* Invalid value. Accepted values are: `on` or `off`."

	if len(a.CommandArgs) != 2 {
		return helpText, nil
	}

	var value bool
	switch a.CommandArgs[1] {
	case settingOn:
		value = true
	case settingOff:
		value = false
	default:
		return helpText, nil
	}

	if a.JiraUser.Settings == nil {
		a.JiraUser.Settings = &UserSettings{}
	}
	a.JiraUser.Settings.Notifications = value
	if err := p.StoreUserInfo(a.Instance, a.MattermostUserId, *a.JiraUser); err != nil {
		return "", errors.WithMessage(err, "Could not store new settings. Please contact your system administrator")
	}

	notifications := "off"
	if a.JiraUser.Settings.Notifications {
		notifications = "on"
	}

	return fmt.Sprintf("Settings updated. Notifications %s.", notifications), nil
}
