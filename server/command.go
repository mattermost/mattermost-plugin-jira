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

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	if command != "/jira" {
		return nil, nil
	}

	switch action {
	case "connect":
		if *p.API.GetConfig().ServiceSettings.SiteURL == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "plugin configuration error."), nil
		}

		resp := getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("[Click here to link your JIRA account.](%s/plugins/%s/oauth/connect2)",
				*p.API.GetConfig().ServiceSettings.SiteURL, manifest.Id))
		return resp, nil
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Command not supported."), nil
}
