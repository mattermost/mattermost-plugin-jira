package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const helpText = "###### Mattermost Jira Plugin - Slash Command Help\n" +
	"* `/jira connect` - Connect your Mattermost account to your Jira account and subscribe to events\n" +
	"* `/jira disconnect` - Disonnect your Mattermost account from your Jira account\n" +
	"* `/jira transition <issue-key> <state>` - Changes the state of a Jira issue.\n" +
	"\nFor system administrators:\n" +
	"* `/jira install cloud <URL>` - connect Mattermost to a cloud Jira instance located at <URL>\n" +
	"* `/jira install server <URL>` - connect Mattermost to a server Jira instance located at <URL>\n" +
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
			Filters: []ActionFunc{RequireCommandMattermostUserId, RequireMattermostSysAdmin},
			Handler: executeInstallServer,
		},
		"instance/add/cloud": &ActionScript{
			Filters: []ActionFunc{RequireCommandMattermostUserId, RequireMattermostSysAdmin},
			Handler: executeInstallCloud,
		},
		"instance/list": &ActionScript{
			Filters: []ActionFunc{RequireCommandMattermostUserId, RequireMattermostSysAdmin},
			Handler: executeInstanceList,
		},
		"instance/select": &ActionScript{
			Filters: []ActionFunc{RequireCommandMattermostUserId, RequireMattermostSysAdmin},
			Handler: executeInstanceSelect,
		},
		"instance/delete": &ActionScript{
			Filters: []ActionFunc{RequireCommandMattermostUserId, RequireMattermostSysAdmin},
			Handler: executeInstanceDelete,
		},
		"webhook": &ActionScript{
			Filters: []ActionFunc{RequireCommandMattermostUserId, RequireMattermostSysAdmin},
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

func executeInstallCloud(a *Action) error {
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

	const addResponseFormat = `
%s has been successfully installed. To finish the configuration, create a new app in your Jira instance following these steps:

1. Navigate to [**Settings > Apps > Manage Apps**](%s/plugins/servlet/upm?source=side_nav_manage_addons).
  - For older versions of Jira, navigate to **Administration > Applications > Add-ons > Manage add-ons**.
2. Click **Settings** at bottom of page, enable development mode, and apply this change.
  - Enabling development mode allows you to install apps that are not from the Atlassian Marketplace.
3. Click **Upload app**.
4. In the **From this URL field**, enter: %s%s
5. Wait for the app to install. Once completed, you should see an "Installed and ready to go!" message.
6. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
7. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.
`

	// TODO What is the exact group membership in Jira required? Site-admins?
	return a.RespondPrintf(addResponseFormat, jiraURL, jiraURL, a.Plugin.GetPluginURL(), routeACJSON)
}

func executeInstallServer(a *Action) error {
	if len(a.CommandArgs) != 1 {
		return executeHelp(a)
	}
	jiraURL := a.CommandArgs[0]

	const addResponseFormat = `` +
		`Server instance has been installed. To finish the configuration, add an Application Link in your Jira instance following these steps:

1. Navigate to **Settings > Applications > Application Links**
2. Enter %s as the application link, then click **Create new link**.
3. In **Configure Application URL** screen, confirm your Mattermost URL is included as the application URL. Ignore any displayed errors and click **Continue**.
4. In **Link Applications** screen, set the following values:
  - **Application Name**: Mattermost
  - **Application Type**: Generic Application
5. Check the **Create incoming link** value, then click **Continue**.
6. In the following **Link Applications** screen, set the following values:
  - **Consumer Key**: %s
  - **Consumer Name**: Mattermost
  - **Public Key**: %s
7. Click **Continue**.
6. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
7. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.
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
	return a.RespondPrintf(addResponseFormat, ji.GetURL(), ji.GetMattermostKey(), pkey)
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
		AutoCompleteDesc: "Available commands: connect, disconnect, transition, install cloud, install server, help",
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
