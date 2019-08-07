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
	"* `/jira connect` - Connect your Mattermost account to your Jira account\n" +
	"* `/jira disconnect` - Disconnect your Mattermost account from your Jira account\n" +
	"* `/jira assign <issue-key> <assignee>` - Change the assignee of a Jira issue\n" +
	"* `/jira create <text (optional)>` - Create a new Issue with 'text' inserted into the description field\n" +
	"* `/jira transition <issue-key> <state>` - Change the state of a Jira issue\n" +
	"* `/jira view <issue-key>` or `/jira <issue-key>` - View a Jira issue\n" +
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
		"view":             executeView,
		"settings":         executeSettings,
		"transition":       executeTransition,
		"assign":           executeAssign,
		"uninstall/cloud":  executeUninstallCloud,
		"uninstall/server": executeUninstallServer,
		"webhook":          executeWebhookURL,
		"info":             executeInfo,
		"help":             commandHelp,
		// "list":             executeList,
		// "instance/select":  executeInstanceSelect,
		// "instance/delete":  executeInstanceDelete,
	},
	defaultHandler: executeJiraDefault,
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
	return p.help(header)
}

func (p *Plugin) help(args *model.CommandArgs) *model.CommandResponse {
	p.postCommandResponse(args, helpText)
	return &model.CommandResponse{}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	if len(args) == 0 || args[0] != "/jira" {
		return p.help(commandArgs), nil
	}
	return jiraCommandHandler.Handle(p, c, commandArgs, args[1:]...), nil
}

func executeConnect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return p.help(header)
	}

	instance, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return p.responsef(header, "There is no Jira instance installed. Please contact your system administrator.")
	}

	jiraUser, err := p.userStore.LoadJIRAUser(instance, header.UserId)
	if err == nil && len(jiraUser.Key()) != 0 {
		return p.responsef(header, "You already have a Jira account linked to your Mattermost account. Please use `/jira disconnect` to disconnect.")
	}

	redirectURL, err := instance.GetUserConnectURL(header.UserId)
	if err != nil {
		return p.responsef(header, "Command failed, please contact your system administrator: %v", err)
	}

	return p.responseRedirect(redirectURL)
}

func executeDisconnect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return p.help(header)
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		p.errorf("executeDisconnect: failed to load current Jira instance: %v", err)
		return p.responsef(header, "Failed to load current Jira instance. Please contact your system administrator.")
	}

	jiraUser, err := p.userStore.LoadJIRAUser(ji, header.UserId)
	if err != nil {
		return p.responsef(header, "Could not complete the **disconnection** request. You do not currently have a Jira account linked to your Mattermost account.")
	}

	err = p.userDisconnect(ji, header.UserId)
	if err != nil {
		return p.responsef(header, "Could not complete the **disconnection** request. Error: %v", err)
	}

	return p.responsef(header, "You have successfully disconnected your Jira account (**%s**).", jiraUser.DisplayName)
}

func executeSettings(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 1 {
		return p.help(header)
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		p.errorf("executeSettings: failed to load current Jira instance: %v", err)
		return p.responsef(header, "Failed to load current Jira instance. Please contact your system administrator.")
	}

	mattermostUserId := header.UserId
	jiraUser, err := p.userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`. %v", err)
	}

	switch args[0] {
	case settingsNotifications:
		return p.settingsNotifications(header, ji, mattermostUserId, jiraUser, args)
	default:
		return p.responsef(header, "Unknown setting.")
	}
}

// executeJiraDefault is the default command if no other command fits. It defaults to help.
func executeJiraDefault(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	return p.help(header)
}

// executeView returns a Jira issue formatted as a slack attachment, or an error message.
func executeView(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return p.responsef(header, "Please specify an issue key in the form `/jira view <issue-key>`.")
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		p.errorf("executeView: failed to load current Jira instance: %v", err)
		return p.responsef(header, "Failed to load current Jira instance. Please contact your system administrator.")
	}

	mattermostUserId := header.UserId
	jiraUser, err := p.userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		// v2.2: try to retrieve the issue anonymously
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`.")
	}

	attachment, err := p.getIssueAsSlackAttachment(ji, jiraUser, strings.ToUpper(args[0]))
	if err != nil {
		return p.responsef(header, err.Error())
	}

	post := &model.Post{
		UserId:    p.getUserID(),
		ChannelId: header.ChannelId,
	}
	post.AddProp("attachments", attachment)

	_ = p.API.SendEphemeralPost(header.UserId, post)

	return &model.CommandResponse{}
}

func executeList(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira list` can only be run by a system administrator.")
	}
	if len(args) != 0 {
		return p.help(header)
	}

	known, err := p.instanceStore.LoadKnownJIRAInstances()
	if err != nil {
		return p.responsef(header, "Failed to load known Jira instances: %v", err)
	}
	if len(known) == 0 {
		return p.responsef(header, "(none installed)\n")
	}

	current, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return p.responsef(header, "Failed to load current Jira instance: %v", err)
	}

	keys := []string{}
	for key := range known {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	text := "Known Jira instances (selected instance is **bold**)\n\n| |URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		ji, err := p.instanceStore.LoadJIRAInstance(key)
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
	return p.responsef(header, text)
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
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira install` can only be run by a system administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}
	jiraURL, err := normalizeInstallURL(args[0])
	if err != nil {
		return p.responsef(header, err.Error())
	}

	// Create an "uninitialized" instance of Jira Cloud that will
	// receive the /installed callback
	err = p.instanceStore.CreateInactiveCloudInstance(jiraURL)
	if err != nil {
		return p.responsef(header, err.Error())
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
	return p.responsef(header, addResponseFormat, jiraURL, jiraURL, p.GetPluginURL(), routeACJSON)
}

func executeInstallServer(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira install` can only be run by a system administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}
	jiraURL, err := normalizeInstallURL(args[0])
	if err != nil {
		return p.responsef(header, err.Error())
	}

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
	err = p.instanceStore.StoreJIRAInstance(ji)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	err = p.StoreCurrentJIRAInstanceAndNotify(ji)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	pkey, err := publicKeyString(p)
	if err != nil {
		return p.responsef(header, "Failed to load public key: %v", err)
	}
	return p.responsef(header, addResponseFormat, p.GetSiteURL(), ji.GetMattermostKey(), pkey)
}

// executeUninstallCloud will uninstall the jira cloud instance if the url matches, and then update all connected
// clients so that their Jira-related menu options are removed.
func executeUninstallCloud(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira uninstall` can only be run by a System Administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}

	jiraURL, err := normalizeInstallURL(args[0])
	if err != nil {
		return p.responsef(header, err.Error())
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return p.responsef(header, "No current Jira instance to uninstall")
	}

	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return p.responsef(header, "The current Jira instance is not a cloud instance")
	}

	if jiraURL != jci.GetURL() {
		return p.responsef(header, "You have entered an incorrect URL. The current Jira instance URL is: `"+jci.GetURL()+"`. Please enter the URL correctly to confirm the uninstall command.")
	}

	err = p.instanceStore.DeleteJiraInstance(jci.GetURL())
	if err != nil {
		return p.responsef(header, "Failed to delete Jira instance "+ji.GetURL())
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

	return p.responsef(header, uninstallInstructions)
}

func executeUninstallServer(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira uninstall` can only be run by a System Administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}

	jiraURL, err := normalizeInstallURL(args[0])
	if err != nil {
		return p.responsef(header, err.Error())
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return p.responsef(header, "No current Jira instance to uninstall")
	}

	jsi, ok := ji.(*jiraServerInstance)
	if !ok {
		return p.responsef(header, "The current Jira instance is not a server instance")
	}

	if jiraURL != jsi.GetURL() {
		return p.responsef(header, "You have entered an incorrect URL. The current Jira instance URL is: `"+jsi.GetURL()+"`. Please enter the URL correctly to confirm the uninstall command.")
	}

	err = p.instanceStore.DeleteJiraInstance(jsi.GetURL())
	if err != nil {
		return p.responsef(header, "Failed to delete Jira instance "+ji.GetURL())
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

	return p.responsef(header, uninstallInstructions)
}

func executeAssign(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {

	if len(args) != 2 {
		return p.responsef(header, "Please specify both an issue key and assignee in the form `/jira assign <issue-key> <assignee>`.")
	}

	issueKey := args[0]
	assignee := args[1]

	msg, err := p.assignJiraIssue(header.UserId, issueKey, assignee)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

func executeTransition(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 2 {
		return p.help(header)
	}
	issueKey := args[0]
	toState := strings.Join(args[1:], " ")

	msg, err := p.transitionJiraIssue(header.UserId, issueKey, toState)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

func executeInfo(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return p.help(header)
	}

	uinfo := getUserInfo(p, header.UserId)

	resp := ""
	switch {
	case uinfo.IsConnected:
		resp = fmt.Sprintf("Connected to Jira %s as %s.\n", uinfo.JIRAURL, uinfo.JIRAUser.DisplayName)
	case uinfo.InstanceInstalled:
		resp = fmt.Sprintf("Jira %s is installed, but you are not connected. Please use `/jira connect`.\n", uinfo.JIRAURL)
	default:
		return p.responsef(header, "No Jira instance installed, please contact your system administrator.")
	}

	resp += fmt.Sprintf("\nJira instance: %q\n", uinfo.JIRAURL)
	for k, v := range uinfo.InstanceDetails {
		resp += fmt.Sprintf(" * %s: %s\n", k, v)
	}

	if uinfo.IsConnected {
		resp += fmt.Sprintf("\nMattermost:\n")
		resp += fmt.Sprintf(" * User ID: %s\n", header.UserId)
		resp += fmt.Sprintf(" * Settings: %+v\n", uinfo.JIRAUser.Settings)

		if uinfo.JIRAUser.Oauth1AccessToken != "" {
			resp += fmt.Sprintf(" * OAuth1a access token: %s\n", uinfo.JIRAUser.Oauth1AccessToken)
			resp += fmt.Sprintf(" * OAuth1a access secret: XXX (%v bytes)\n", len(uinfo.JIRAUser.Oauth1AccessSecret))
		}

		juser := uinfo.JIRAUser.User
		resp += fmt.Sprintf("\nJira user: %s\n", juser.DisplayName)
		resp += fmt.Sprintf(" * Self: %s\n", juser.Self)
		resp += fmt.Sprintf(" * AccountID: %s\n", juser.AccountID)
		resp += fmt.Sprintf(" * Name (deprecated): %s\n", juser.Name)
		resp += fmt.Sprintf(" * Key (deprecated): %s\n", juser.Key)
		resp += fmt.Sprintf(" * EmailAddress: %s\n", juser.EmailAddress)
		resp += fmt.Sprintf(" * Active: %v\n", juser.Active)
		resp += fmt.Sprintf(" * TimeZone: %v\n", juser.TimeZone)
		resp += fmt.Sprintf(" * ApplicationKeys: %s\n", juser.ApplicationKeys)
	}
	return p.responsef(header, resp)
}

func executeWebhookURL(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira webhook` can only be run by a system administrator.")
	}
	if len(args) != 0 {
		return p.help(header)
	}

	u, err := p.GetWebhookURL(header.TeamId, header.ChannelId)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	return p.responsef(header, "Please use the following URL to set up a Jira webhook: %v", u)
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "jira",
		DisplayName:      "Jira",
		Description:      "Integration with Jira.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, create, transition, view, settings, install cloud/server, uninstall cloud/server, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.getUserID(),
		ChannelId: args.ChannelId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) responsef(commandArgs *model.CommandArgs, format string, args ...interface{}) *model.CommandResponse {
	p.postCommandResponse(commandArgs, fmt.Sprintf(format, args...))
	return &model.CommandResponse{}
}

func (p *Plugin) responseRedirect(redirectURL string) *model.CommandResponse {
	return &model.CommandResponse{
		GotoLocation: redirectURL,
	}
}

func executeInstanceSelect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return p.help(header)
	}
	instanceKey := args[0]
	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		known, loadErr := p.instanceStore.LoadKnownJIRAInstances()
		if loadErr != nil {
			return p.responsef(header, "Failed to load known Jira instances: %v", err)
		}
		if num < 1 || int(num) > len(known) {
			return p.responsef(header, "Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	ji, err := p.instanceStore.LoadJIRAInstance(instanceKey)
	if err != nil {
		return p.responsef(header, "Failed to load Jira instance %s: %v", instanceKey, err)
	}
	err = p.StoreCurrentJIRAInstanceAndNotify(ji)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	return executeInfo(p, c, header)
}

func executeInstanceDelete(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return p.help(header)
	}
	instanceKey := args[0]

	known, err := p.instanceStore.LoadKnownJIRAInstances()
	if err != nil {
		return p.responsef(header, "Failed to load known JIRA instances: %v", err)
	}
	if len(known) == 0 {
		return p.responsef(header, "There are no instances to delete.\n")
	}

	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		if num < 1 || int(num) > len(known) {
			return p.responsef(header, "Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	// Remove the instance
	err = p.instanceStore.DeleteJiraInstance(instanceKey)
	if err != nil {
		return p.responsef(header, "failed to delete Jira instance %s: %v", instanceKey, err)
	}

	// if that was our only instance, just respond with an empty list.
	if len(known) == 1 {
		return executeList(p, c, header)
	}
	return executeInstanceSelect(p, c, header, "1")
}
