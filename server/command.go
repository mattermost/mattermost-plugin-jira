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

func getCommandResponse(responseType, text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     JIRA_USERNAME,
		IconURL:      JIRA_ICON_URL,
		Type:         model.POST_DEFAULT,
	}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(commandArgs.Command)
	if len(split) < 2 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Command not supported. %v", len(split))), nil
	}
	command := split[0]
	if command != "/jira" {
		return nil, nil
	}
	action := split[1]

	switch action {
	case "connect":
		if *p.API.GetConfig().ServiceSettings.SiteURL == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "plugin configuration error."), nil
		}

		resp := getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			// fmt.Sprintf("[Click here to link your JIRA account.](%s/plugins/%s/oauth/connect)",
			fmt.Sprintf("[Click here to link your JIRA account.](%s/plugins/%s/user-connect)",
				*p.API.GetConfig().ServiceSettings.SiteURL, manifest.Id))
		return resp, nil

	case "disconnect":
		if *p.API.GetConfig().ServiceSettings.SiteURL == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "plugin configuration error."), nil
		}

		resp := getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			// fmt.Sprintf("[Click here to link your JIRA account.](%s/plugins/%s/oauth/connect)",
			fmt.Sprintf("[Click here to unlink your JIRA account.](%s/plugins/%s/user-disconnect)",
				*p.API.GetConfig().ServiceSettings.SiteURL, manifest.Id))
		return resp, nil

	case "instances":
		known, err := p.LoadKnownJIRAInstances()
		if err != nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
		}

		current, err := p.LoadCurrentJIRAInstance()
		if err != nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
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

		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Command not supported."), nil
}
