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

func responsef(format string, args ...interface{}) *model.CommandResponse {
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
		return responsef("Invalid syntax. Must be at least /jira action."), nil
	}
	command := split[0]
	if command != "/jira" {
		return nil, nil
	}
	action := split[1]
	split = split[2:]

	switch action {
	case "connect":
		if p.GetSiteURL() == "" {
			return responsef("plugin configuration error."), nil
		}

		return responsef("[Click here to link your JIRA account.](%s/user-connect)",
			p.GetPluginURL()), nil

	case "disconnect":
		if p.GetSiteURL() == "" {
			return responsef("plugin configuration error."), nil
		}

		return responsef("[Click here to unlink your JIRA account.](%s/user-disconnect)",
			p.GetPluginURL()), nil

	case "instance":
		if len(split) < 1 {
			return responsef("/jira instance [add,list,select]"), nil
		}
		verb := split[0]
		split = split[1:]

		switch verb {
		case "list":
			known, err := p.LoadKnownJIRAInstances()
			if err != nil {
				return responsef("couldn't load known JIRA instances: %v", err), nil
			}

			current, err := p.LoadCurrentJIRAInstance()
			if err != nil {
				return responsef("couldn't load current JIRA instance: %v", err), nil
			}

			text := ""
			for key, typ := range known {
				if key == current.GetURL() {
					text += "**"
				}
				text += key + " - " + typ
				if key == current.GetURL() {
					text += "**"
				}
				text += "\n"
			}

			if text == "" {
				text = "(none installed)\n"
			}

			return responsef(text), nil

		case "add":
			if len(split) < 2 {
				return responsef("/jira instance add {type} {URL}"), nil
			}
			typ := split[0]
			if typ != JIRATypeServer {
				return responsef(`only type "server" supported by /jira add`), nil
			}
			jiraURL := split[1]

			ji := NewJIRAServerInstance(p, jiraURL)
			err := p.StoreJIRAInstance(ji, true)
			if err != nil {
				return responsef("failed to store JIRA instance %s: %v", jiraURL, err), nil
			}

			return responsef("Added and selected %s (type %s).", jiraURL, typ), nil

		case "select":
			if len(split) < 1 {
				return responsef("/jira instance select {URL}"), nil
			}
			jiraURL := split[0]

			ji, err := p.LoadJIRAInstance(jiraURL)
			if err != nil {
				return responsef("failed to load JIRA instance %s: %v", jiraURL, err), nil
			}
			err = p.StoreJIRAInstance(ji, true)
			if err != nil {
				return responsef("failed to store JIRA instance %s: %v", jiraURL, err), nil
			}

			return responsef("Now using JIRA at %s", jiraURL), nil
		}
	}

	return responsef("Command %v is not supported.", action), nil
}
