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
	"  * `/jira instance delete <number or URL>` - Delete a Jira instance. If the current instance is deleted, the first remaining instance is selected as current.\n" +
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
		return executeInstanceList(p, c)
	case "add":
		return executeInstanceAdd(p, c, args...)
	case "select":
		return executeInstanceSelect(p, c, args...)
	case "delete":
		return executeInstanceDelete(p, c, args...)
	}
	return responsef("Please specify a parameter in the form `/jira instance [add,list,select]")
}

func executeInstanceList(p *Plugin, c *plugin.Context) *model.CommandResponse {
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
	text := "Known JIRA instances (selected instance is **bold**)\n\n| |URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		ji, err := p.LoadJIRAInstance(key)
		if err != nil {
			text += fmt.Sprintf("|%v|%s|error: %v|\n", i+1, key, err)
			continue
		}
		details := ""
		for k, v := range ji.GetDisplayDetails() {
			details += fmt.Sprintf("%s:%s, ", k, v)
		}
		if len(details) > len(", ") {
			details = details[:len(details)-2]
		} else {
			details = ji.GetType()
		}
		format := "|%v|%s|%s|\n"
		if key == current.GetURL() {
			format = "| **%v** | **%s** |%s|\n"
		}
		text += fmt.Sprintf(format, i+1, key, details)
	}
	return responsef(text)
}

const addResponseFormat = `Instance has been added. You need to add an Application Link to it in Jira now.
1. Click %s, login as an admin.
2. Navigate to (Jira) Settings > Applications > Application Links.
3. Enter %s, anc click "Create new link".
4. In "Configure Application URL" screen ignore any errors, click "Continue".
5. In "Link Applications":
  - Application Name: Mattermost
  - Application Type: Generic Application
  - IMPORTANT: Check "Create incoming link"
6. In "Link Applications", pt. 2:
  - Consumer Key: %s
  - Consumer Name: Mattermost
  - Public Key: %s
`

func executeInstanceAdd(p *Plugin, c *plugin.Context, args ...string) *model.CommandResponse {
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
		err := p.StoreJIRAInstanceSetCurrent(ji)
		if err != nil {
			return responsef("Failed to store JIRA instance %s: %v", jiraURL, err)
		}

		pkey, err := publicKeyString(p)
		if err != nil {
			return responsef("Failed to load public key: %v", err)
		}
		return responsef(addResponseFormat, ji.GetURL(), p.GetSiteURL(), ji.GetMattermostKey(), pkey)

	case JIRATypeCloud:
		// TODO the exact group membership in JIRA?
		return responsef(`As an admin, upload an application from %s/%s. The link can be found in **JIRA Settings > Applications > Manage**`,
			p.GetPluginURL(), routeACJSON)
	}

	return responsef("Please specify a parameter in the form `/jira instance add server {URL}` or `/jira instance add cloud`")
}

func executeInstanceSelect(p *Plugin, c *plugin.Context, args ...string) *model.CommandResponse {
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
			return responsef("Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
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
	err = p.StoreCurrentJIRAInstance(ji)
	if err != nil {
		return responsef("Failed to store Jira instance %s: %v", instanceKey, err)
	}

	return executeInstanceList(p, c)
}

func executeInstanceDelete(p *Plugin, c *plugin.Context, args ...string) *model.CommandResponse {
	if len(args) < 1 {
		return responsef("/jira instance delete {URL|#}")
	}
	instanceKey := args[0]

	known, err := p.LoadKnownJIRAInstances()
	if err != nil {
		return responsef("Failed to load known JIRA instances: %v", err)
	}

	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		if num < 1 || int(num) > len(known) {
			return responsef("Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	// Remove the instance
	err = p.DeleteJiraInstance(instanceKey)
	if err != nil {
		return responsef("failed to delete JIRA instance %s: %v", instanceKey, err)
	}

	return executeInstanceSelect(p, c, "1")
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
