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
	"* `/jira transition <issue-key> <state>` - Changes the state of a Jira issue.\n" +
	"* `/jira instance [add/list/select/delete]` - Manage connected Jira instances\n" +
	"  * `add server <URL>` - Add a Jira Server instance\n" +
	"  * `add cloud <URL>` - Add a Jira Cloud instance\n" +
	"  * `list` - List known Jira instances\n" +
	"  * `select <number or URL>` - Select a known instance as current\n" +
	"  * `delete <number or URL>` - Delete a known instance, select the first remaining as the current\n" +
	"* `/jira webhook` - Display a Jira webhook URL customized for the current team/channel\n" +
	""

type CommandHandlerFunc func(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse

type CommandHandler struct {
	handlers       map[string]CommandHandlerFunc
	defaultHandler CommandHandlerFunc
}

var jiraCommandHandler = CommandHandler{
	handlers: map[string]CommandHandlerFunc{
		"instance/add/server":    executeInstanceAddServer,
		"instance/add/cloud":     executeInstanceAddCloud,
		"instance/list":          executeInstanceList,
		"instance/select":        executeInstanceSelect,
		"instance/delete":        executeInstanceDelete,
		"settings/notifications": executeSettingsNotifications,
		"webhook":                executeWebhookURL,
		"webhook/url":            executeWebhookURL,
		"transition":             executeTransition,
		"connect":                executeConnect,
		"disconnect":             executeDisconnect,
	},
	defaultHandler: commandHelp,
}

func (ch CommandHandler) Handle(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	for n := len(args); n > 0; n-- {
		h := ch.handlers[strings.Join(args[:n], "/")]
		if h != nil {
			return h(p, c, header, args[n:]...)
		}
	}
	return ch.defaultHandler(p, c, header, args...)
}

func commandHelp(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	return help()
}

func help() *model.CommandResponse {
	return responsef(helpText)
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	if len(args) == 0 || args[0] != "/jira" {
		return help(), nil
	}
	return jiraCommandHandler.Handle(p, c, commandArgs, args[1:]...), nil
}

func executeConnect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return help()
	}
	return responsef("[Click here to link your Jira account.](%s%s)",
		p.GetPluginURL(), routeUserConnect)
}

func executeDisconnect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return help()
	}
	return responsef("[Click here to unlink your Jira account.](%s%s)",
		p.GetPluginURL(), routeUserDisconnect)
}

func executeSettingsNotifications(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return help()
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return responsef("Failed to load current Jira instance: %v. Please contact your system administrator.", err)
	}

	mattermostUserId := header.UserId
	jiraUser, err := p.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return responsef("Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}

	return responsef("todo... JiraUser: %v", jiraUser)
}

func executeInstanceList(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return help()
	}
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

func executeInstanceAddServer(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return help()
	}
	jiraURL := args[0]

	const addResponseFormat = `` +
		`Instance has been added. You need to add an Application Link to it in Jira now.
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
	ji := NewJIRAServerInstance(p, jiraURL)
	err := p.StoreJIRAInstance(ji)
	if err != nil {
		return responsef(err.Error())
	}
	err = p.StoreCurrentJIRAInstance(ji)
	if err != nil {
		return responsef(err.Error())
	}

	pkey, err := publicKeyString(p)
	if err != nil {
		return responsef("Failed to load public key: %v", err)
	}
	return responsef(addResponseFormat, ji.GetURL(), p.GetSiteURL(), ji.GetMattermostKey(), pkey)
}

func executeInstanceAddCloud(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return help()
	}
	jiraURL := args[0]

	// Create an "uninitialized" instance of Jira Cloud that will
	// receive the /installed callback
	err := p.CreateInactiveCloudInstance(jiraURL)
	if err != nil {
		return responsef(err.Error())
	}
	// TODO What is the exact group membership in Jira required? Site-admins?
	return responsef(`%s has been successfully added. To complete the installation:
* navigate to [**Jira > Applications > Manage**](%s/plugins/servlet/upm?source=side_nav_manage_addons)
* click "Upload app"
* enter the following URL: %s%s`,
		jiraURL, jiraURL, p.GetPluginURL(), routeACJSON)
}

func executeInstanceSelect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return help()
	}
	instanceKey := args[0]
	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		known, loadErr := p.LoadKnownJIRAInstances()
		if loadErr != nil {
			return responsef("Failed to load known Jira instances: %v", err)
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
		return responsef(err.Error())
	}

	return executeInstanceList(p, c, header)
}

func executeInstanceDelete(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return help()
	}
	instanceKey := args[0]

	known, err := p.LoadKnownJIRAInstances()
	if err != nil {
		return responsef("Failed to load known JIRA instances: %v", err)
	}
	if len(known) == 0 {
		return responsef("There are no instances to delete.\n")
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
		return responsef("failed to delete Jira instance %s: %v", instanceKey, err)
	}

	// if that was our only instance, just respond with an empty list.
	if len(known) == 1 {
		return executeInstanceList(p, c, header)
	}
	return executeInstanceSelect(p, c, header, "1")
}

func executeTransition(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 2 {
		return help()
	}
	issueKey := args[0]
	toState := strings.Join(args[1:], " ")

	if err := p.transitionJiraIssue(header.UserId, issueKey, toState); err != nil {
		return responsef("%v", err)
	}

	return responsef("Transition completed.")
}

func executeWebhookURL(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return help()
	}

	u, err := p.GetWebhookURL(header.TeamId, header.ChannelId)
	if err != nil {
		return responsef(err.Error())
	}
	return responsef("Please use the following URL to set up a Jira webhook: %v", u)
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "jira",
		DisplayName:      "Jira",
		Description:      "Integration with Jira.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, transition, instance, help",
		AutoCompleteHint: "[command]",
	}
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
