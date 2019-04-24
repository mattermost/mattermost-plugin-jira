package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const helpText = "###### Mattermost JIRA Plugin - Slash Command Help\n" +
	"* `/jira connect` - Connect your Mattermost account to your JIRA account\n" +
	"* `/jira disconnect` - Disonnect your Mattermost account to your JIRA account\n" +
	"* `/jira instance` - Manage JIRA instances connected to Mattermost\n" +
	"  * `list` - List known JIRA instances\n" +
	"  * `select <key or number>` - Select a known instance as current\n" +
	"  * `add server <URL>` - Add a JIRA Server instance\n" +
	"  * `add cloud` - Add a JIRA Cloud instance\n" +
	""

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "jira",
		DisplayName:      "JIRA",
		Description:      "Integration with JIRA.",
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
	return responsef("[Click here to link your JIRA account.](%s/%s)",
		p.GetPluginURL(), routeUserConnect)
}

func executeDisconnect(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	return responsef("[Click here to unlink your JIRA account.](%s/%s)",
		p.GetPluginURL(), routeUserDisconnect)
}

func executeInstance(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	if len(args) < 1 {
		return responsef("Usage: /jira instance [add,list,select,delete]")
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
	return responsef("Usage: /jira instance [add,list,select,delete]")
}

func executeInstanceList(p *Plugin, c *plugin.Context, args []string) *model.CommandResponse {
	known, err := p.LoadKnownJIRAInstances()
	if err != nil {
		return responsef("Failed to load known JIRA instances: %v", err)
	}
	if len(known) == 0 {
		return responsef("(none installed)\n")
	}

	current, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return responsef("Failed to load current JIRA instance: %v", err)
	}

	keys := []string{}
	for key := range known {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	text := "Known JIRA instances (selected instance is **bold**)\n\n| |Key|Type|\n|--|--|--|\n"
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
		return responsef("Usage: `/jira instance add server {URL}` or `/jira instance add cloud`")
	}
	typ := args[0]

	switch typ {
	case JIRATypeServer:
		if len(args) < 2 {
			return responsef("Usage: `/jira instance add server {URL}`")
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
		return responsef(`As an admin, upload an application from %s/%s. The link can be found in "JIRA Settings/Applications/Manage"`,
			p.GetPluginURL(), routeACJSON)
	}

	return responsef("Usage: `/jira instance add server {URL}` or `/jira instance add cloud`")
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
		return responsef("failed to load JIRA instance %s: %v", instanceKey, err)
	}
	err = p.StoreJIRAInstance(ji, true)
	if err != nil {
		return responsef("failed to store JIRA instance %s: %v", instanceKey, err)
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
