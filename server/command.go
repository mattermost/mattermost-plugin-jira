package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const helpText = "###### Mattermost Jira Plugin - Slash Command Help\n" +
	"* `/jira connect` - Connect your Mattermost account to your Jira account\n" +
	"* `/jira disconnect` - Disconnect your Mattermost account from your Jira account\n" +
	"* `/jira create <text (optional)>` - Create a new Issue with 'text' inserted into the description field.\n" +
	"* `/jira transition <issue-key> <state>` - Change the state of a Jira issue\n" +
	"* `/jira settings [setting] [value]` - Update your user settings\n" +
	"  * [setting] can be `notifications`\n" +
	"  * [value] can be `on` or `off`\n" +

	"\n###### For System Administrators:\n" +
	"Install:\n" +
	"* `/jira install cloud <URL>` - Connect Mattermost to a Jira Cloud instance located at <URL>\n" +
	"* `/jira install server <URL>` - Connect Mattermost to a Jira Server or Data Center instance located at <URL>\n" +
	"Uninstall:\n" +
	"* `/jira uninstall cloud <URL>` - Disconnect Mattermost from a Jira Cloud instance located at <URL>\n" +
	"* `/jira uninstall server <URL>` - Disconnect Mattermost from a Jira Server or Data Center instance located at <URL>\n" +
	""

// Available settings
const (
	settingsNotifications = "notifications"
)

type CommandHandlerFunc func(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse

type CommandHandler struct {
	handlers       map[string]CommandHandlerFunc
	defaultHandler CommandHandlerFunc
}

var jiraCommandHandler = CommandHandler{
	handlers: map[string]CommandHandlerFunc{
		"connect":          executeConnect,
		"disconnect":       executeDisconnect,
		"install/cloud":    executeInstallCloud,
		"install/server":   executeInstallServer,
		"settings":         executeSettings,
		"transition":       executeTransition,
		"uninstall/cloud":  executeUninstallCloud,
		"uninstall/server": executeUninstallServer,
		//"webhook":        executeWebhookURL,
		//"webhook/url":    executeWebhookURL,
		//"list":        executeList,
		//"instance/select":     executeInstanceSelect,
		//"instance/delete":     executeInstanceDelete,
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
	return responsef("[Click here to link your Jira account](%s%s)",
		p.GetPluginURL(), routeUserConnect)
}

func executeDisconnect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return help()
	}
	return responsef("[Click here to unlink your Jira account](%s%s)",
		p.GetPluginURL(), routeUserDisconnect)
}

func executeSettings(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 1 {
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

	switch args[0] {
	case settingsNotifications:
		return p.settingsNotifications(ji, mattermostUserId, jiraUser, args)
	default:
		return responsef("Unknown setting.")
	}
}

func executeList(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return responsef("%v", err)
	}
	if !authorized {
		return responsef("`/jira list` can only be run by a system administrator.")
	}
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

func authorizedSysAdmin(p *Plugin, userId string) (bool, error) {
	user, err := p.API.GetUser(userId)
	if err != nil {
		return false, err
	}
	if !strings.Contains(user.Roles, "system_admin") {
		return false, nil
	}
	return true, nil
}

func executeInstallCloud(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return responsef("%v", err)
	}
	if !authorized {
		return responsef("`/jira install` can only be run by a system administrator.")
	}
	if len(args) != 1 {
		return help()
	}
	jiraURL := args[0]

	// Create an "uninitialized" instance of Jira Cloud that will
	// receive the /installed callback
	err = p.CreateInactiveCloudInstance(jiraURL)
	if err != nil {
		return responsef(err.Error())
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
	return responsef(addResponseFormat, jiraURL, jiraURL, p.GetPluginURL(), routeACJSON)
}

func executeInstallServer(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return responsef("%v", err)
	}
	if !authorized {
		return responsef("`/jira install` can only be run by a system administrator.")
	}
	if len(args) != 1 {
		return help()
	}
	jiraURL := args[0]

	const addResponseFormat = `` +
		`Server instance has been installed. To finish the configuration, add an Application Link in your Jira instance following these steps:

1. Navigate to **Settings > Applications > Application Links**
2. Enter %s as the application link, then click **Create new link**.
3. In **Configure Application URL** screen, confirm your Mattermost URL is entered as the "New URL". Ignore any displayed errors and click **Continue**.
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
	ji := NewJIRAServerInstance(p, jiraURL)
	err = p.StoreJIRAInstance(ji)
	if err != nil {
		return responsef(err.Error())
	}
	err = p.StoreCurrentJIRAInstanceAndNotify(ji)
	if err != nil {
		return responsef(err.Error())
	}

	pkey, err := publicKeyString(p)
	if err != nil {
		return responsef("Failed to load public key: %v", err)
	}
	return responsef(addResponseFormat, p.GetSiteURL(), ji.GetMattermostKey(), pkey)
}

// executeUninstallCloud will uninstall the jira cloud instance if the url matches, and then update all connected
// clients so that their Jira-related menu options are removed.
func executeUninstallCloud(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return responsef("%v", err)
	}
	if !authorized {
		return responsef("`/jira uninstall` can only be run by a System Administrator.")
	}
	if len(args) != 1 {
		return help()
	}
	jiraURL := args[0]

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return responsef("No current Jira instance to uninstall")
	}

	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return responsef("The current Jira instance is not a cloud instance")
	}

	if jiraURL != jci.GetURL() {
		return responsef("You have entered an incorrect URL. The current Jira instance URL is: `" + jci.GetURL() + "`. Please enter the URL correctly to confirm the uninstall command.")
	}

	err = p.DeleteJiraInstance(jci.GetURL())
	if err != nil {
		return responsef("Failed to delete Jira instance " + ji.GetURL())
	}

	// Notify users we have uninstalled an instance
	p.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": false,
		},
		&model.WebsocketBroadcast{},
	)

	const uninstallInstructions = `Jira instance successfully disconnected. Go to **Settings > Apps > Manage Apps** to remove the application in your Jira Cloud instance.`

	return responsef(uninstallInstructions)
}

func executeUninstallServer(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return responsef("%v", err)
	}
	if !authorized {
		return responsef("`/jira uninstall` can only be run by a System Administrator.")
	}
	if len(args) != 1 {
		return help()
	}
	jiraURL := args[0]

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return responsef("No current Jira instance to uninstall")
	}

	jsi, ok := ji.(*jiraServerInstance)
	if !ok {
		return responsef("The current Jira instance is not a server instance")
	}

	if jiraURL != jsi.GetURL() {
		return responsef("You have entered an incorrect URL. The current Jira instance URL is: `" + jsi.GetURL() + "`. Please enter the URL correctly to confirm the uninstall command.")
	}

	err = p.DeleteJiraInstance(jsi.GetURL())
	if err != nil {
		return responsef("Failed to delete Jira instance " + ji.GetURL())
	}

	// Notify users we have uninstalled an instance
	p.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": false,
		},
		&model.WebsocketBroadcast{},
	)

	const uninstallInstructions = `Jira instance successfully disconnected. Go to **Settings > Applications > Application Links** to remove the application in your Jira Server or Data Center instance.`

	return responsef(uninstallInstructions)
}

func executeTransition(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 2 {
		return help()
	}
	issueKey := args[0]
	toState := strings.Join(args[1:], " ")

	msg, err := p.transitionJiraIssue(header.UserId, issueKey, toState)
	if err != nil {
		return responsef("%v", err)
	}

	return responsef(msg)
}

func executeWebhookURL(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return responsef("%v", err)
	}
	if !authorized {
		return responsef("`/jira webhook` can only be run by a system administrator.")
	}
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
		AutoCompleteDesc: "Available commands: connect, disconnect, create, transition, settings, install cloud/server, uninstall cloud/server, help",
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

// Uncomment if needed for development: (and uncomment the command handlers above)
//
//func executeInstanceSelect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
//	if len(args) != 1 {
//		return help()
//	}
//	instanceKey := args[0]
//	num, err := strconv.ParseUint(instanceKey, 10, 8)
//	if err == nil {
//		known, loadErr := p.LoadKnownJIRAInstances()
//		if loadErr != nil {
//			return responsef("Failed to load known Jira instances: %v", err)
//		}
//		if num < 1 || int(num) > len(known) {
//			return responsef("Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
//		}
//
//		keys := []string{}
//		for key := range known {
//			keys = append(keys, key)
//		}
//		sort.Strings(keys)
//		instanceKey = keys[num-1]
//	}
//
//	ji, err := p.LoadJIRAInstance(instanceKey)
//	if err != nil {
//		return responsef("Failed to load Jira instance %s: %v", instanceKey, err)
//	}
//	err = p.StoreCurrentJIRAInstanceAndNotify(ji)
//	if err != nil {
//		return responsef(err.Error())
//	}
//
//	return executeList(p, c, header)
//}
//
//func executeInstanceDelete(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
//	if len(args) != 1 {
//		return help()
//	}
//	instanceKey := args[0]
//
//	known, err := p.LoadKnownJIRAInstances()
//	if err != nil {
//		return responsef("Failed to load known JIRA instances: %v", err)
//	}
//	if len(known) == 0 {
//		return responsef("There are no instances to delete.\n")
//	}
//
//	num, err := strconv.ParseUint(instanceKey, 10, 8)
//	if err == nil {
//		if num < 1 || int(num) > len(known) {
//			return responsef("Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
//		}
//
//		keys := []string{}
//		for key := range known {
//			keys = append(keys, key)
//		}
//		sort.Strings(keys)
//		instanceKey = keys[num-1]
//	}
//
//	// Remove the instance
//	err = p.DeleteJiraInstance(instanceKey)
//	if err != nil {
//		return responsef("failed to delete Jira instance %s: %v", instanceKey, err)
//	}
//
//	// if that was our only instance, just respond with an empty list.
//	if len(known) == 1 {
//		return executeList(p, c, header)
//	}
//	return executeInstanceSelect(p, c, header, "1")
//}
