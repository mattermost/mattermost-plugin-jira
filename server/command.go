package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const COMMAND_HELP = `* |/jira connect| - Connect your Mattermost account to your JIRA account
`

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "jira",
		DisplayName:      "JIRA",
		Description:      "Integration with JIRA.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect",
		AutoCompleteHint: "[command]",
	}
}

func ephf(format string, args ...interface{}) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf(format, args...),
		Username:     JIRA_USERNAME,
		IconURL:      JIRA_ICON_URL,
		Type:         model.POST_DEFAULT,
	}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(commandArgs.Command)
	if len(split) < 2 {
		return ephf("Invalid syntax. Must be at least /jira action."), nil
	}
	command := split[0]
	if command != "/jira" {
		return nil, nil
	}
	action := split[1]
	split = split[2:]

	switch action {
	case "connect":
		if p.externalURL() == "" {
			return ephf("plugin configuration error."), nil
		}

		return ephf("[Click here to link your JIRA account.](%s/plugins/%s/user-connect)",
			p.externalURL(), manifest.Id), nil

	case "disconnect":
		if p.externalURL() == "" {
			return ephf("plugin configuration error."), nil
		}

		return ephf("[Click here to unlink your JIRA account.](%s/plugins/%s/user-disconnect)",
			p.externalURL(), manifest.Id), nil

	case "instances":
		known, err := p.LoadKnownJIRAInstances()
		if err != nil {
			return ephf("couldn't load known JIRA instances: %v", err), nil
		}

		current, err := p.LoadCurrentJIRAInstance()
		if err != nil {
			return ephf("couldn't load current JIRA instance: %v", err), nil
		}

		text := ""
		for key, typ := range known {
			if key == current.Key {
				text += "*"
			} else {
				text += " "
			}
			text += key + ": " + typ + "\n"
		}

		if text == "" {
			text = "(none installed)"
		}
		return ephf(text), nil

	case "add":
		if len(split) < 2 {
			return ephf("/jira add {type} {URL}"), nil
		}
		typ := split[0]
		if typ != JIRAServerType {
			return ephf(`only type "server" supported by /jira add`), nil
		}
		jiraURL := split[1]

		// Create or overwrite the instance record, also store it
		// as current
		rsaKey, err := p.EnsureRSAKey()
		if err != nil {
			return ephf("failed to obtain an RSA key: %v", err), nil
		}
		jiraInstance := NewJIRAServerInstance(jiraURL, p.externalURL(), rsaKey)
		err = p.StoreJIRAInstance(jiraInstance, true)
		if err != nil {
			return ephf("failed to store JIRA instance %s: %v", jiraURL, err), nil
		}

		return ephf("Added and selected %s (type %s).", jiraURL, typ), nil
	}

	return ephf("Command %v is not supported.", action), nil
}
