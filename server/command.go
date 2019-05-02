package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const helpText = "###### Mattermost Jira Plugin - Slash Command Help\n" +
	"* `/jira connect` - Connect your Mattermost account to your Jira account and subscribe to events\n" +
	"* `/jira disconnect` - Disonnect your Mattermost account from your Jira account\n" +
	"* `/jira instance [add/list/select]` - Manage connected Jira instances\n" +
	"  * `/jira instance add server <URL>` - Connect a Jira Server instance to Mattermost\n" +
	"  * `/jira instance add cloud` - Connect a Jira Cloud instance to Mattermost\n" +
	"  * `/jira instance list` - List connected Jira instances\n" +
	"  * `/jira instance select <number or URL>` - Select the active Jira instance. At most one active Jira instance is currently supported\n" +
	""

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "jira",
		DisplayName:      "Jira",
		Description:      "Integration with Jira.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	if len(args) < 2 {
		return responsef("Invalid syntax. Must be at least /jira action."), nil
	}
	action := args[1]
	args = args[2:]

	switch action {
	case "help":
		return responsef(helpText), nil
	case "connect":
		return executeConnect(p, c, args), nil
	case "disconnect":
		return executeDisconnect(p, c, args), nil
	case "instance":
		return executeInstance(p, c, args), nil
	}

	return responsef("Action %v is not supported.", action), nil
}

func executeConnect(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	return responsef("[Click here to link your Jira account.](%s/%s)",
		p.GetPluginURL(), routeUserConnect)
}

func executeDisconnect(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	return responsef("[Click here to unlink your Jira account.](%s/%s)",
		p.GetPluginURL(), routeUserDisconnect)
}

func executeInstance(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	if len(args) < 1 {
		return responsef("Please specify a parameter in the form `/jira instance [add,list,select]")
	}
	action := args[0]
	args = args[1:]

	switch action {
	case "list":
		return executeInstanceList(p, c, args)
	case "add":
		return executeInstanceAdd(p, c, args)
	case "select":
		return executeInstanceSelect(p, c, args)
	}
	return responsef("Please specify a parameter in the form `/jira instance [add,list,select]")
}

func executeInstanceList(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	known, err := p.LoadKnownJIRAInstances()
	if err != nil {
		return responsef("Failed to load known Jira instances: %v", err)
	}
	if len(known) == 0 {
		return responsef("(none installed)\n")
	}

	current, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return responsef("Failed to load current Jira instance: %v", err)
	}

	keys := []string{}
	for key := range known {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	text := "Known Jira instances (selected instance is **bold**)\n\n| |URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		typ := known[key]
		if key == current.GetURL() {
			key = "**" + key + "**"
			typ = "**" + typ + "**"
		}
		text += fmt.Sprintf("|%v|%s|%s|\n", i+1, key, typ)
	}
	return responsef(text)
}

func executeInstanceAdd(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	if len(args) < 1 {
		return responsef("Please specify a parameter in the form `/jira instance add server {URL}` or `/jira instance add cloud`")
	}
	typ := args[0]

	switch typ {
	case JIRATypeServer:
		if len(args) < 2 {
			return responsef("Please specify the server URL in the form `/jira instance add server {URL}`")
		}
		jiraURL := args[1]

		ji := NewJIRAServerInstance(p, jiraURL)
		err := p.StoreJIRAInstance(ji, true)
		if err != nil {
			return responsef("Failed to store JIRA instance %s: %v", jiraURL, err)
		}
		return executeInstanceList(p, c, nil)

	case JIRATypeCloud:
		// TODO the exact group membership in JIRA?
		return responsef(`As an admin, upload an application from %s/%s. The link can be found in **JIRA Settings > Applications > Manage**`,
			p.GetPluginURL(), routeACJSON)
	}

	return responsef("Please specify a parameter in the form `/jira instance add server {URL}` or `/jira instance add cloud`")
}

func executeInstanceSelect(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	if len(args) < 1 {
		return responsef("/jira instance select {URL|#} ")
	}
	instanceKey := args[0]
	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		known, loadErr := p.LoadKnownJIRAInstances()
		if loadErr != nil {
			return responsef("Failed to load known JIRA instances: %v", err)
		}
		if num < 1 || int(num) > len(known) {
			return responsef("Wrong instance number %v, must be 1-%v\n", num, len(known))
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	ji, err := p.LoadJIRAInstance(instanceKey)
	if err != nil {
		return responsef("Failed to load Jira instance %s: %v", instanceKey, err)
	}
	err = p.StoreJIRAInstance(ji, true)
	if err != nil {
		return responsef("Failed to store Jira instance %s: %v", instanceKey, err)
	}

	return executeInstanceList(p, c, args)
}

func responsef(format string, args ...interface{}) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf(format, args...),
		Username:     PluginMattermostUsername,
		IconURL:      PluginIconURL,
		Type:         model.POST_DEFAULT,
	}
}
