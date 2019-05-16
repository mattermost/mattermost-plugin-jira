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

var commandRouter = ActionRouter{
	DefaultRouteHandler: executeHelp,
	Log: []ActionFunc{
		func(a *Action) error {
			a.Plugin.debugf("command: %q", a.CommandHeader.Command)
			return nil
		},
	},
	RouteHandlers: map[string]*ActionScript{
		"instance/add/server": &ActionScript{
			Handler: executeInstanceAddServer,
		},
		"instance/add/cloud": &ActionScript{
			Handler: executeInstanceAddCloud,
		},
		"instance/list": &ActionScript{
			Handler: executeInstanceList,
		},
		"instance/select": &ActionScript{
			Handler: executeInstanceSelect,
		},
		"instance/delete": &ActionScript{
			Handler: executeInstanceDelete,
		},
		"webhook": &ActionScript{
			Handler: executeWebhookURL,
		},
		"webhook/url": &ActionScript{
			Handler: executeWebhookURL,
		},
		"transition": &ActionScript{
			Filters: []ActionFunc{RequireCommandMattermostUserId, RequireJiraClient},
			Handler: executeTransition,
		},
		"connect": &ActionScript{
			Handler: executeConnect,
		},
		"disconnect": &ActionScript{
			Handler: executeDisconnect,
		},
	},
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	action := NewAction(p, c)
	action.CommandHeader = commandArgs
	action.CommandArgs = args[1:]

	if len(args) == 0 || args[0] != "/jira" {
		executeHelp(action)
		return action.CommandResponse, nil
	}
	args = args[1:]

	scriptKey := ""
	for n := len(args); n > 0; n-- {
		key := strings.Join(args[:n], "/")
		if commandRouter.RouteHandlers[key] != nil {
			action.CommandArgs = args[n:]
			scriptKey = key
			break
		}
	}

	commandRouter.Run(scriptKey, action)
	return action.CommandResponse, nil
}

func executeHelp(a *Action) error {
	return a.RespondPrintf(helpText)
}

func executeConnect(a *Action) error {
	if len(a.CommandArgs) != 0 {
		return executeHelp(a)
	}
	return a.RespondPrintf("[Click here to link your Jira account.](%s%s)",
		a.Plugin.GetPluginURL(), routeUserConnect)
}

func executeDisconnect(a *Action) error {
	if len(a.CommandArgs) != 0 {
		return executeHelp(a)
	}
	return a.RespondPrintf("[Click here to unlink your Jira account.](%s%s)",
		a.Plugin.GetPluginURL(), routeUserDisconnect)
}

func executeInstanceList(a *Action) error {
	if len(a.CommandArgs) != 0 {
		return executeHelp(a)
	}
	known, err := a.Plugin.LoadKnownJIRAInstances()
	if err != nil {
		return a.RespondError(0, err)
	}
	if len(known) == 0 {
		return a.RespondPrintf("(none installed)\n")
	}

	current, err := a.Plugin.LoadCurrentJIRAInstance()
	if err != nil {
		return a.RespondError(0, err)
	}

	keys := []string{}
	for key := range known {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	text := "Known Jira instances (selected instance is **bold**)\n\n| |URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		ji, err := a.Plugin.LoadJIRAInstance(key)
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
	return a.RespondPrintf(text)
}

func executeInstanceAddServer(a *Action) error {
	if len(a.CommandArgs) != 1 {
		return executeHelp(a)
	}
	jiraURL := a.CommandArgs[0]

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
	ji := NewJIRAServerInstance(jiraURL, a.Plugin.GetPluginKey())
	err := a.Plugin.StoreJIRAInstance(ji)
	if err != nil {
		return a.RespondError(0, err)
	}
	err = a.Plugin.StoreCurrentJIRAInstance(ji)
	if err != nil {
		return a.RespondError(0, err)
	}

	pkey, err := publicKeyString(a.Plugin)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf(addResponseFormat,
		ji.GetURL(), a.Plugin.GetSiteURL(), ji.GetMattermostKey(), pkey)
}

func executeInstanceAddCloud(a *Action) error {
	if len(a.CommandArgs) != 1 {
		return executeHelp(a)
	}
	jiraURL := a.CommandArgs[0]

	// Create an "uninitialized" instance of Jira Cloud that will
	// receive the /installed callback
	err := a.Plugin.CreateInactiveCloudInstance(jiraURL)
	if err != nil {
		return a.RespondError(0, err)
	}
	// TODO What is the exact group membership in Jira required? Site-admins?
	return a.RespondPrintf(`%s has been successfully added. To complete the installation:
* navigate to [**Jira > Applications > Manage**](%s/plugins/servlet/upm?source=side_nav_manage_addons)
* click "Upload app"
* enter the following URL: %s%s`,
		jiraURL, jiraURL, a.Plugin.GetPluginURL(), routeACJSON)
}

func executeInstanceSelect(a *Action) error {
	if len(a.CommandArgs) != 1 {
		return executeHelp(a)
	}
	instanceKey := a.CommandArgs[0]
	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		known, loadErr := a.Plugin.LoadKnownJIRAInstances()
		if loadErr != nil {
			return a.RespondError(0, err)
		}
		if num < 1 || int(num) > len(known) {
			return a.RespondError(0, nil,
				"Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	ji, err := a.Plugin.LoadJIRAInstance(instanceKey)
	if err != nil {
		return a.RespondError(0, err)
	}
	err = a.Plugin.StoreCurrentJIRAInstance(ji)
	if err != nil {
		return a.RespondError(0, err)
	}

	a.CommandArgs = []string{}
	return executeInstanceList(a)
}

func executeInstanceDelete(a *Action) error {
	if len(a.CommandArgs) != 1 {
		return executeHelp(a)
	}
	instanceKey := a.CommandArgs[0]

	known, err := a.Plugin.LoadKnownJIRAInstances()
	if err != nil {
		return a.RespondError(0, err)
	}
	if len(known) == 0 {
		return a.RespondError(0, nil,
			"There are no instances to delete.\n")
	}

	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		if num < 1 || int(num) > len(known) {
			return a.RespondError(0, nil,
				"Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	// Remove the instance
	err = a.Plugin.DeleteJiraInstance(instanceKey)
	if err != nil {
		return a.RespondError(0, err)
	}

	// if that was our only instance, just respond with an empty list.
	if len(known) == 1 {
		a.CommandArgs = []string{}
		return executeInstanceList(a)
	}

	// Select instance #1
	a.CommandArgs = []string{"1"}
	return executeInstanceSelect(a)
}

func executeTransition(a *Action) error {
	if len(a.CommandArgs) < 2 {
		return executeHelp(a)
	}
	issueKey := a.CommandArgs[0]
	toState := strings.Join(a.CommandArgs[1:], " ")

	if err := transitionJiraIssue(a, issueKey, toState); err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf("Transition completed.")
}

func executeWebhookURL(a *Action) error {
	if len(a.CommandArgs) != 0 {
		return executeHelp(a)
	}

	u, err := a.Plugin.GetWebhookURL(a.CommandHeader.TeamId, a.CommandHeader.ChannelId)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf("Please use the following URL to set up a Jira webhook: %v", u)
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
func commandResponse(format string, args ...interface{}) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf(format, args...),
		Username:     PluginMattermostUsername,
		IconURL:      PluginIconURL,
		Type:         model.POST_DEFAULT,
	}
}
