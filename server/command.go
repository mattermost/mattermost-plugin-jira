package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin" 

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

const helpTextHeader = "###### Mattermost Jira Plugin - Slash Command Help\n"

const commonHelpText = "\n* `/jira connect` - Connect your Mattermost account to your Jira account\n" +
	"* `/jira disconnect` - Disconnect your Mattermost account from your Jira account\n" +
	"* `/jira assign <issue-key> <assignee>` - Change the assignee of a Jira issue\n" +
	"* `/jira unassign <issue-key>` - Unassign the Jira issue\n" +
	"* `/jira create <text (optional)>` - Create a new Issue with 'text' inserted into the description field\n" +
	"* `/jira transition <issue-key> <state>` - Change the state of a Jira issue\n" +
	"* `/jira stats` - Poll basic statistics about the Jira plug-in.  Recevie performace metrics and authenticated users between Jira/Mattermost\n" +	
	"* `/jira info` - Retrieve information about the current user and the Jira plug-in\n" +
	"* `/jira help` - Launch the Jira plug-in command line help syntax\n" +	
	"* `/jira webhook` - Execute the Jira plug-in webhook. The Jira plugin gets sent a stream of pre-configured events from the Jira server\n" +		
	"* `/jira subscribe` - Configure the Jira notifications sent to this channel\n" +
	"* `/jira view <issue-key>` - View the details of a specific Jira issue\n" +
	"* `/jira settings [setting] [value]` - Update your user settings\n" +
	"  * [setting] can be `notifications`\n" +
	"  * [value] can be `on` or `off`\n"

const sysAdminHelpText = "\n###### For System Administrators:\n" +
	"Install:\n" +
	"* `/jira install cloud <URL>` - Connect Mattermost to a Jira Cloud instance located at <URL>\n" +
	"* `/jira install server <URL>` - Connect Mattermost to a Jira Server or Data Center instance located at <URL>\n" +
	"Uninstall:\n" +
	"* `/jira uninstall cloud <URL>` - Disconnect Mattermost from a Jira Cloud instance located at <URL>\n" +
	"* `/jira uninstall server <URL>` - Disconnect Mattermost from a Jira Server or Data Center instance located at <URL>\n" +
	"* `/jira subscribe list` - List of Jira Notification subscription rules across all channels\n"

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
		"connect":            executeConnect,
		"disconnect":         executeDisconnect,
		"install/cloud":      executeInstallCloud,
		"install/server":     executeInstallServer,
		"view":               executeView,
		"settings":           executeSettings,
		"transition":         executeTransition,
		"assign":             executeAssign,
		"unassign":           executeUnassign,
		"uninstall":          executeUninstall,
		"webhook":            executeWebhookURL,
		"stats":              executeStats,
		"info":               executeInfo,
		"help":               commandHelp,
		"subscribe/list":     executeSubscribeList,
		"debug/stats/reset":  executeDebugStatsReset,
		"debug/stats/save":   executeDebugStatsSave,
		"debug/stats/expvar": executeDebugStatsExpvar,
		"debug/workflow":     executeDebugWorkflow,
		// "debug/instance/list":   executeDebugInstanceList,
		// "debug/instance/select": executeDebugInstanceSelect,
		// "debug/instance/delete": executeDebugInstanceDelete,
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
	authorized, _ := authorizedSysAdmin(p, args.UserId)

	helpText := helpTextHeader
	jiraAdminAdditionalHelpText := p.getConfig().JiraAdminAdditionalHelpText

	// Check if JIRA admin has provided additional help text to be shown up along with regular output
	if jiraAdminAdditionalHelpText != "" {
		helpText += "    " + jiraAdminAdditionalHelpText
	}

	helpText += commonHelpText

	if authorized {
		helpText += sysAdminHelpText
	}

	p.postCommandResponse(args, helpText)
	return &model.CommandResponse{}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	err := p.CheckSiteURL()
	if err != nil {
		return p.responsef(commandArgs, err.Error()), nil
	}
	args := strings.Fields(commandArgs.Command)
	if len(args) == 0 || args[0] != "/jira" {
		return p.help(commandArgs), nil
	}
	return jiraCommandHandler.Handle(p, c, commandArgs, args[1:]...), nil
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

	return p.responsef(header, "[Click here to link your Jira account](%s%s)",
		p.GetPluginURL(), routeUserConnect)
}

func executeSettings(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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

	if len(args) == 0 {
		return p.responsef(header, "Current settings:\n%s", jiraUser.Settings.String())
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

func executeDebugInstanceList(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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

func executeSubscribeList(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira subscribe list` can only be run by a system administrator.")
	}

	msg, err := p.listChannelSubscriptions(header.TeamId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

func authorizedSysAdmin(p *Plugin, userId string) (bool, error) {
	user, appErr := p.API.GetUser(userId)
	if appErr != nil {
		return false, appErr
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
	jiraURL, err := utils.NormalizeInstallURL(p.GetSiteURL(), args[0])
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
	jiraURL, err := utils.NormalizeInstallURL(p.GetSiteURL(), args[0])
	if err != nil {
		return p.responsef(header, err.Error())
	}
	isJiraCloudURL, err := utils.IsJiraCloudURL(jiraURL)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	if isJiraCloudURL {
		return p.responsef(header, "The Jira URL you provided looks like a Jira Cloud URL - install it with:\n```\n/jira install cloud %s\n```", jiraURL)
	}

	const addResponseFormat = `` +
		`Server instance has been installed. To finish the configuration, add an Application Link in your Jira instance following these steps:

1. Navigate to [**Settings > Applications > Application Links**](%s/plugins/servlet/applinks/listApplicationLinks)
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
	return p.responsef(header, addResponseFormat, jiraURL, p.GetSiteURL(), ji.GetMattermostKey(), pkey)
}

// executeUninstall will uninstall the jira instance if the url matches, and then update all connected clients
// so that their Jira-related menu options are removed.
func executeUninstall(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira uninstall` can only be run by a System Administrator.")
	}
	if len(args) != 2 {
		return p.help(header)
	}

	jiraURL, err := utils.NormalizeInstallURL(p.GetSiteURL(), args[1])
	if err != nil {
		return p.responsef(header, err.Error())
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return p.responsef(header, "No current Jira instance to uninstall")
	}

	var ok bool
	switch args[0] {
	case "cloud":
		_, ok = ji.(*jiraCloudInstance)
	case "server":
		_, ok = ji.(*jiraServerInstance)
	default:
		return p.help(header)
	}

	if !ok {
		return p.responsef(header, fmt.Sprintf("The current Jira instance is not a %s instance", args[0]))
	}

	if jiraURL != ji.GetURL() {
		return p.responsef(header, `You have entered an incorrect URL. The current Jira instance URL is %s. Please enter the URL correctly to confirm the uninstall command.`, ji.GetURL())
	}

	err = p.instanceStore.DeleteJiraInstance(ji.GetURL())
	if err != nil {
		return p.responsef(header, "Failed to delete Jira instance "+ji.GetURL())
	}

	// Notify users we have uninstalled an instance
	p.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": false,
			"instance_type":      "",
		},
		&model.WebsocketBroadcast{},
	)

	uninstallInstructions := `Jira instance successfully uninstalled. Navigate to [**your app management URL**](%s) in order to remove the application from your Jira instance.`
	return p.responsef(header, uninstallInstructions, ji.GetManageAppsURL())
}

func executeUnassign(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 1 {
		return p.responsef(header, "Please specify an issue key in the form `/jira unassign <issue-key>`.")
	}
	issueKey := strings.ToUpper(args[0])

	msg, err := p.unassignJiraIssue(header.UserId, issueKey)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

func executeAssign(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 2 {
		return p.responsef(header, "Please specify an issue key and an assignee search string, in the form `/jira assign <issue-key> <assignee>`.")
	}
	issueKey := strings.ToUpper(args[0])
	userSearch := strings.Join(args[1:], " ")

	msg, err := p.assignJiraIssue(header.UserId, issueKey, userSearch)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

func executeTransition(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 2 {
		return p.help(header)
	}
	issueKey := strings.ToUpper(args[0])
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

	resp := fmt.Sprintf("Mattermost Jira plugin version: %s, "+
		"[%s](https://github.com/mattermost/mattermost-plugin-jira/commit/%s), built %s\n",
		manifest.Version, BuildHashShort, BuildHash, BuildDate)

	switch {
	case uinfo.IsConnected:
		resp += fmt.Sprintf("Connected to Jira %s as %s.\n", uinfo.JIRAURL, uinfo.JIRAUser.DisplayName)
	case uinfo.InstanceInstalled:
		resp += fmt.Sprintf("Jira %s is installed, but you are not connected. Please [connect](%s/%s).\n",
			uinfo.JIRAURL, p.GetPluginURL(), routeUserConnect)
	default:
		return p.responsef(header, resp+"\nNo Jira instance installed, please contact your system administrator.")
	}

	resp += fmt.Sprintf("\nJira:\n")

	for k, v := range uinfo.InstanceDetails {
		resp += fmt.Sprintf(" * %s: %s\n", k, v)
	}

	bullet := func(cond bool, k string, v interface{}) string {
		if !cond {
			return ""
		}
		return fmt.Sprintf(" * %s: %v\n", k, v)
	}

	sbullet := func(k, v string) string {
		return bullet(v != "", k, v)
	}

	if uinfo.IsConnected {
		juser := uinfo.JIRAUser.User
		resp += sbullet("User", juser.DisplayName)
		resp += sbullet("Self", juser.Self)
		resp += sbullet("AccountID", juser.AccountID)
		resp += sbullet("Name", juser.Name)
		resp += sbullet("Key", juser.Key)
		resp += sbullet("EmailAddress", juser.EmailAddress)
		resp += bullet(juser.Active, "Active", juser.Active)
		resp += sbullet("TimeZone", juser.TimeZone)
		resp += bullet(len(juser.ApplicationKeys) > 0, "ApplicationKeys", juser.ApplicationKeys)

		resp += fmt.Sprintf("\nMattermost:\n")

		resp += sbullet("Site URL", p.GetSiteURL())
		resp += sbullet("User ID", header.UserId)

		resp += fmt.Sprintf(" * Settings: %+v", uinfo.JIRAUser.Settings)

		if uinfo.JIRAUser.Oauth1AccessToken != "" {
			resp += sbullet("OAuth1a access token", uinfo.JIRAUser.Oauth1AccessToken)
			resp += sbullet("OAuth1a access secret (length)", strconv.Itoa(len(uinfo.JIRAUser.Oauth1AccessSecret)))
		}
	}
	return p.responsef(header, resp)
}

func executeStats(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	return executeStatsImpl(p, c, commandArgs, false, args...)
}

func executeDebugStatsExpvar(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	return executeStatsImpl(p, c, commandArgs, true, args...)
}

func executeDebugWorkflow(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	return p.responsef(commandArgs, "Workflow Store:\n %v", p.workflowTriggerStore)
}

func executeStatsImpl(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, useExpvar bool, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, commandArgs.UserId)
	if err != nil {
		return p.responsef(commandArgs, "%v", err)
	}
	if !authorized {
		return p.responsef(commandArgs, "`/jira stats` can only be run by a system administrator.")
	}
	if len(args) < 1 {
		return p.help(commandArgs)
	}
	resp := fmt.Sprintf("Mattermost Jira plugin version: %s, "+
		"[%s](https://github.com/mattermost/mattermost-plugin-jira/commit/%s), built %s\n",
		manifest.Version, BuildHashShort, BuildHash, BuildDate)

	pattern := strings.Join(args, " ")
	print := expvar.PrintExpvars
	if !useExpvar {
		var stats *expvar.Stats
		var keys []string
		stats, keys, err = p.consolidatedStoredStats()
		if err != nil {
			return p.responsef(commandArgs, "%v", err)
		}
		resp += fmt.Sprintf("Consolidated from stored keys `%s`\n", keys)
		print = stats.PrintConsolidated
	}

	rstats, err := print(pattern)
	if err != nil {
		return p.responsef(commandArgs, "%v", err)
	}

	return p.responsef(commandArgs, resp+rstats)
}

func executeDebugStatsReset(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, commandArgs.UserId)
	if err != nil {
		return p.responsef(commandArgs, "%v", err)
	}
	if !authorized {
		return p.responsef(commandArgs, "`/jira stats` can only be run by a system administrator.")
	}
	if len(args) != 0 {
		return p.help(commandArgs)
	}

	err = p.debugResetStats()
	if err != nil {
		return p.responsef(commandArgs, err.Error())
	}
	return p.responsef(commandArgs, "Reset stats")
}

func executeDebugStatsSave(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, commandArgs.UserId)
	if err != nil {
		return p.responsef(commandArgs, "%v", err)
	}
	if !authorized {
		return p.responsef(commandArgs, "`/jira stats` can only be run by a system administrator.")
	}
	if len(args) != 0 {
		return p.help(commandArgs)
	}
	stats := p.getConfig().stats
	if stats == nil {
		return p.responsef(commandArgs, "No stats to save")
	}
	p.saveStats()
	return p.responsef(commandArgs, "Saved stats")
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
		AutoCompleteDesc: "Available commands: connect, assign, disconnect, create, transition, view, subscribe, settings, install cloud/server, uninstall cloud/server, help",
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

func executeDebugInstanceSelect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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

func executeDebugInstanceDelete(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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
		return executeDebugInstanceList(p, c, header)
	}
	return executeDebugInstanceSelect(p, c, header, "1")
}
