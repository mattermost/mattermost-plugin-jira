package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const helpTextHeader = "###### Mattermost Jira Plugin - Slash Command Help\n"

// <><> TODO Update command Help
const commonHelpText = "\n* `/jira connect` - Connect your Mattermost account to your Jira account\n" +
	"* `/jira disconnect` - Disconnect your Mattermost account from your Jira account\n" +
	"* `/jira assign <issue-key> <assignee>` - Change the assignee of a Jira issue\n" +
	"* `/jira unassign <issue-key>` - Unassign the Jira issue\n" +
	"* `/jira create <text (optional)>` - Create a new Issue with 'text' inserted into the description field\n" +
	"* `/jira transition <issue-key> <state>` - Change the state of a Jira issue\n" +
	"* `/jira info` - Display information about the current user and the Jira plug-in\n" +
	"* `/jira help` - Launch the Jira plugin command line help syntax\n" +
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
	"* `/jira stats` - Display usage statistics\n" +
	"* `/jira webhook` -  Show the Mattermost webhook to receive JQL queries\n" +
	"* `/jira subscribe` - Configure the Jira notifications sent to this channel\n" +
	"* `/jira subscribe list` - Display all the the subscription rules setup across all the channels and teams on your Mattermost instance\n"

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
		"connect":                 executeConnect,
		"disconnect":              executeDisconnect,
		"install/cloud":           executeInstanceInstallCloud,
		"install/server":          executeInstanceInstallServer,
		"view":                    executeView,
		"settings":                executeSettings,
		"transition":              executeTransition,
		"assign":                  executeAssign,
		"unassign":                executeUnassign,
		"uninstall":               executeInstanceUninstall,
		"webhook":                 executeWebhookURL,
		"stats":                   executeStats,
		"info":                    executeInfo,
		"help":                    commandHelp,
		"subscribe/list":          executeSubscribeList,
		"debug/stats/reset":       executeDebugStatsReset,
		"debug/stats/save":        executeDebugStatsSave,
		"debug/stats/expvar":      executeDebugStatsExpvar,
		"debug/workflow":          executeDebugWorkflow,
		"debug/clean-instances":   executeDebugCleanInstances,
		"debug/migrate-instances": executeDebugMigrateInstances,
		"instance/install/cloud":  executeInstanceInstallCloud,
		"instance/install/server": executeInstanceInstallServer,
		"instance/list":           executeInstanceList,
		"instance/default":        executeInstanceDefault,
		"instance/uninstall":      executeInstanceUninstall,
		"instance/settings":       executeSettings,
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
	instanceID, args, err := p.parseCommandFlagInstanceID(args)
	if err != nil {
		return p.responsef(header, "Failed to identify a Jira instance. Error: "+err.Error())
	}
	if len(args) != 0 {
		return p.help(header)
	}
	disconnected, err := p.DisconnectUser(instanceID, types.ID(header.UserId))
	if err == ErrConnectionNotFound {
		return p.responsef(header, "Could not complete the **disconnection** request. You do not currently have a Jira account linked to your Mattermost account.")
	}
	if err != nil {
		return p.responsef(header, "Could not complete the **disconnection** request. Error: %v", err)
	}
	return p.responsef(header, "You have successfully disconnected your Jira account (**%s**).", disconnected.DisplayName)
}

func executeConnect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	instanceID, args, err := p.parseCommandFlagInstanceID(args)
	if err != nil {
		return p.responsef(header, "Failed to identify a Jira instance. Error: "+err.Error())
	}
	if len(args) != 0 {
		return p.help(header)
	}

	link := routeUserConnect
	if instanceID != "" {
		conn, err := p.userStore.LoadConnection(instanceID, types.ID(header.UserId))
		if err == nil && len(conn.JiraAccountID()) != 0 {
			return p.responsef(header,
				"You already have a Jira account linked to your Mattermost account from %s. Please use `/jira disconnect --instance=%s` to disconnect.",
				instanceID, instanceID)
		}

		link = p.pathWithInstance(link, instanceID)
	}

	return p.responsef(header, "[Click here to link your Jira account](%s%s)",
		p.GetPluginURL(), link)
}

func executeSettings(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	instanceID, args, err := p.parseCommandFlagInstanceID(args)
	if err != nil {
		return p.responsef(header, "Failed to load Jira instance. Please contact your system administrator. Error: %v.", err)
	}

	mattermostUserID := types.ID(header.UserId)
	conn, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`. Error: %v.", err)
	}

	if len(args) == 0 {
		return p.responsef(header, "Current settings:\n%s", conn.Settings.String())
	}

	switch args[0] {
	case settingsNotifications:
		return p.settingsNotifications(header, instanceID, mattermostUserID, conn, args)
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
	instance, _, err := p.parseCommandFlagInstance(args)
	if err != nil {
		return p.responsef(header, "Failed to identify a Jira instance. Error: "+err.Error())
	}
	if len(args) != 1 {
		return p.responsef(header, "Please specify an issue key in the form `/jira view <issue-key>`.")
	}

	mattermostUserID := types.ID(header.UserId)
	issueID := args[0]

	conn, err := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
	if err != nil {
		// TODO: try to retrieve the issue anonymously
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`.")
	}

	attachment, err := p.getIssueAsSlackAttachment(instance, conn, strings.ToUpper(issueID))
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

func executeInstanceList(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return p.responsef(header, "Failed to load known Jira instances: %v", err)
	}
	if instances.IsEmpty() {
		return p.responsef(header, "(none installed)\n")
	}

	keys := []string{}
	for _, key := range instances.IDs() {
		keys = append(keys, key.String())
	}
	sort.Strings(keys)
	text := "Known Jira instances (selected instance is **bold**)\n\n| |URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		instanceID := types.ID(key)
		instance, err := p.instanceStore.LoadInstance(instanceID)
		if err != nil {
			text += fmt.Sprintf("|%v|%s|error: %v|\n", i+1, key, err)
			continue
		}
		details := ""
		for k, v := range instance.GetDisplayDetails() {
			details += fmt.Sprintf("%s:%s, ", k, v)
		}
		if len(details) > len(", ") {
			details = details[:len(details)-2]
		} else {
			details = string(instance.Common().Type)
		}
		format := "|%v|%s|%s|\n"
		if instances.Get(instanceID).IsDefault {
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
	instanceID, args, err := p.parseCommandFlagInstanceID(args)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	msg, err := p.listChannelSubscriptions(instanceID, header.TeamId)
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

func executeInstanceInstallCloud(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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
	if strings.Contains(jiraURL, "http:") {
		jiraURL = strings.Replace(jiraURL, "http:", "https:", -1)
		return p.responsef(header, "`/jira install cloud` requires a secure connection (HTTPS). Please run the following command:\n```\n/jira install cloud %s\n```", jiraURL)
	}

	// Create an "uninitialized" instance of Jira Cloud that will
	// receive the /installed callback
	err = p.instanceStore.CreateInactiveCloudInstance(types.ID(jiraURL))
	if err != nil {
		return p.responsef(header, err.Error())
	}

	u, err := p.GetWebhookURL(header.TeamId, header.ChannelId)
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

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://mattermost.gitbook.io/plugin-jira) for troubleshooting help.
Jira webhook URL: %s
`

	// TODO What is the exact group membership in Jira required? Site-admins?
	return p.responsef(header, addResponseFormat, jiraURL, jiraURL, p.GetPluginURL(), p.pathWithInstance(routeACJSON, types.ID(jiraURL)), u)
}

func executeInstanceInstallServer(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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

	u, err := p.GetWebhookURL(header.TeamId, header.ChannelId)
	if err != nil {
		return p.responsef(header, err.Error())
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

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://mattermost.gitbook.io/plugin-jira) for troubleshooting help.
Jira webhook URL: %s
`
	instance := newServerInstance(p, jiraURL)
	err = p.InstallInstance(instance)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	pkey, err := publicKeyString(p)
	if err != nil {
		return p.responsef(header, "Failed to load public key: %v", err)
	}
	return p.responsef(header, addResponseFormat, jiraURL, p.GetSiteURL(), instance.GetMattermostKey(), pkey, u)
}

// executeUninstall will uninstall the jira instance if the url matches, and then update all connected clients
// so that their Jira-related menu options are removed.
func executeInstanceUninstall(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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

	instanceType := InstanceType(args[0])
	instanceURL := args[1]

	id, err := utils.NormalizeInstallURL(p.GetSiteURL(), instanceURL)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	uninstalled, err := p.UninstallInstance(types.ID(id), instanceType)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	// Notify users we have uninstalled an instance
	p.API.PublishWebSocketEvent(
		websocketEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": false,
			"instance_type":      "",
		},
		&model.WebsocketBroadcast{},
	)

	u, err := p.GetWebhookURL(header.TeamId, header.ChannelId)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	uninstallInstructions := `` +
		`Jira instance successfully uninstalled. Navigate to [**your app management URL**](%s) in order to remove the application from your Jira instance.
Don't forget to remove Jira-side webhook from URL: %s'
`
	return p.responsef(header, uninstallInstructions, uninstalled.GetManageAppsURL(), u)
}

func executeUnassign(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	instance, args, err := p.parseCommandFlagInstance(args)
	if err != nil {
		return p.responsef(header, "Failed to identify a Jira instance. Error: "+err.Error())
	}

	if len(args) < 1 {
		return p.responsef(header, "Please specify an issue key in the form `/jira unassign <issue-key>`.")
	}
	issueKey := strings.ToUpper(args[0])

	msg, err := p.UnassignIssue(instance, types.ID(header.UserId), issueKey)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	return p.responsef(header, msg)
}

func executeAssign(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	instance, args, err := p.parseCommandFlagInstance(args)
	if err != nil {
		return p.responsef(header, "Failed to identify a Jira instance. Error: "+err.Error())
	}

	if len(args) < 2 {
		return p.responsef(header, "Please specify an issue key and an assignee search string, in the form `/jira assign <issue-key> <assignee>`.")
	}
	issueKey := strings.ToUpper(args[0])
	userSearch := strings.Join(args[1:], " ")

	msg, err := p.AssignIssue(instance, types.ID(header.UserId), issueKey, userSearch)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

// TODO should transition command post to channel? Options?
func executeTransition(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 2 {
		return p.help(header)
	}
	issueKey := strings.ToUpper(args[0])
	toState := strings.Join(args[1:], " ")

	msg, err := p.TransitionIssue(&InTransitionIssue{
		mattermostUserID: types.ID(header.UserId),
		IssueKey:         issueKey,
		ToState:          toState,
	})
	if err != nil {
		return p.responsef(header, err.Error())
	}

	return p.responsef(header, msg)
}

func executeInfo(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 0 {
		return p.help(header)
	}
	mattermostUserID := types.ID(header.UserId)
	bullet := func(cond bool, k string, v interface{}) string {
		if !cond {
			return ""
		}
		return fmt.Sprintf(" * %s: %v\n", k, v)
	}
	sbullet := func(k, v string) string {
		return bullet(v != "", k, v)
	}
	connectionBullet := func(ic *InstanceCommon, connection *Connection) string {
		switch ic.Type {
		case CloudInstanceType:
			return sbullet(ic.InstanceID.String(), fmt.Sprintf("Cloud, connected as **%s** (AccountID: %s)",
				connection.User.DisplayName,
				connection.User.AccountID))
		case ServerInstanceType:
			return sbullet(ic.InstanceID.String(), fmt.Sprintf("Server, connected as **%s** (Name:%s, Key:%s, EmailAddress:%s)",
				connection.User.DisplayName,
				connection.User.Name,
				connection.User.Key,
				connection.User.EmailAddress))
		}
		return ""
	}

	info, err := p.GetUserInfo(mattermostUserID)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	resp := fmt.Sprintf("Mattermost Jira plugin version: %s, "+
		"[%s](https://github.com/mattermost/mattermost-plugin-jira/commit/%s), built %s.\n",
		manifest.Version, BuildHashShort, BuildHash, BuildDate)

	resp += sbullet("Mattermost site URL", p.GetSiteURL())
	resp += sbullet("Mattermost user ID", mattermostUserID.String())

	switch {
	case info.IsConnected:
		resp += fmt.Sprintf("###### Connected to %v Jira instances:\n", info.User.ConnectedInstances.Len())
	case info.Instances.Len() > 0:
		resp += fmt.Sprintf("Jira is installed, but you are not connected. Please [connect](%s%s).\n",
			p.GetPluginURL(), routeUserConnect)
	default:
		return p.responsef(header, resp+"\nNo Jira instances installed, please contact your system administrator.")
	}

	if info.IsConnected {
		for _, instanceID := range info.User.ConnectedInstances.IDs() {
			connection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
			if err != nil {
				return p.responsef(header, err.Error())
			}

			resp += connectionBullet(info.User.ConnectedInstances.Get(instanceID), connection)
			resp += fmt.Sprintf("   * Settings: %+v\n", connection.Settings)
		}
	}

	orphans := ""
	if !info.Instances.IsEmpty() {
		resp += fmt.Sprintf("\n###### Available Jira instances:\n")
		for _, instanceID := range info.Instances.IDs() {
			ic := info.Instances.Get(instanceID)
			if ic.IsDefault {
				resp += sbullet(instanceID.String(), fmt.Sprintf("%s, **default**", ic.Type))
			} else {
				resp += sbullet(instanceID.String(), fmt.Sprintf("%s", ic.Type))
			}
		}

		for _, instanceID := range info.Instances.IDs() {
			if info.IsConnected && info.User.ConnectedInstances.Contains(instanceID) {
				continue
			}
			connection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
			if err != nil {
				if err == ErrUserNotFound {
					continue
				}
				return p.responsef(header, err.Error())
			}

			orphans += connectionBullet(info.Instances.Get(instanceID), connection)
		}
	}
	if orphans != "" {
		resp += fmt.Sprintf("###### Orphant Jira connections:\n%s", orphans)
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

func executeDebugCleanInstances(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira list` can only be run by a system administrator.")
	}
	_ = p.API.KVDelete(keyInstances)
	return p.responsef(header, "Deleted instances\n")
}

func executeDebugMigrateInstances(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira list` can only be run by a system administrator.")
	}

	err = p.instanceStore.MigrateV2Instances()
	if err != nil {
		return p.responsef(header, "Failed to migrated instances: %v\n", err)
	}
	return p.responsef(header, "Migrated instances\n")
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
		AutoCompleteDesc: "Available commands: connect, assign, disconnect, create, transition, info, view, settings, help",
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

func executeInstanceDefault(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) != 1 {
		return p.help(header)
	}
	instanceID := types.ID(args[0])

	err := p.StoreDefaultInstance(instanceID)
	if err != nil {
		return p.responsef(header, "Failed to set default Jira instance %s: %v", instanceID, err)
	}

	return p.responsef(header, "%s is set as the default Jira instance", instanceID)
}

func (p *Plugin) parseCommandFlagInstanceID(args []string) (types.ID, []string, error) {
	value := ""
	remaining := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--instance") {
			remaining = append(remaining, arg)
			continue
		}
		if value != "" {
			return "", nil, errors.New("--instance may not be specified multiple times")
		}
		str := arg[len("--instance"):]

		// --instance=X
		if strings.HasPrefix(str, "=") {
			value = str[1:]
			continue
		}

		// --instance X
		if i == len(args)-1 {
			return "", nil, errors.New("--instance requires a value")
		}
		i++
		value = args[i]
	}

	id, err := p.ResolveInstanceURL(value)
	if err != nil {
		return "", nil, err
	}
	return id, remaining, nil
}

func (p *Plugin) parseCommandFlagInstance(args []string) (Instance, []string, error) {
	instanceID, args, err := p.parseCommandFlagInstanceID(args)
	if err != nil {
		return nil, nil, err
	}

	// already subject to defaults, so load directly
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return nil, nil, err
	}
	return instance, args, nil
}
