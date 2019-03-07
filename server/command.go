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
		AutoCompleteDesc: "Available commands: connect",
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
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Command not supported."), nil
	}
	command := split[0]
	if command != "/jira" {
		return nil, nil
	}
	action := split[1]
	args := split[2:]

	switch action {
	case "connect":
		if *p.API.GetConfig().ServiceSettings.SiteURL == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "plugin configuration error."), nil
		}

		resp := getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("[Click here to link your JIRA account.](%s/plugins/%s/oauth/connect)",
				*p.API.GetConfig().ServiceSettings.SiteURL, manifest.Id))
		return resp, nil

	case "subscribe":
		if len(args) < 0 {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "---- TODO SUBSCRIBE HELP ---- "), nil
		}

		err := p.loadSecurityContext()
		if err != nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
		}

		jirac, err := p.getJIRAClientForServer()
		if err != nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
		}

	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Command not supported."), nil
}
