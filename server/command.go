// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/command"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/flow"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const commandTrigger = "jira"

var jiraCommandHandler = CommandHandler{
	handlers: map[string]CommandHandlerFunc{
		"assign":                       executeAssign,
		"connect":                      executeConnect,
		"disconnect":                   executeDisconnect,
		"help":                         executeHelp,
		"me":                           executeMe,
		"about":                        executeAbout,
		"install/cloud":                executeInstanceInstallCloud,
		"install/cloud-oauth":          executeInstanceInstallCloudOAuth,
		"install/server":               executeInstanceInstallServer,
		"instance/alias":               executeInstanceAlias,
		"instance/unalias":             executeInstanceUnalias,
		"instance/connect":             executeConnect,
		"instance/disconnect":          executeDisconnect,
		"instance/install/cloud":       executeInstanceInstallCloud,
		"instance/install/cloud-oauth": executeInstanceInstallCloudOAuth,
		"instance/install/server":      executeInstanceInstallServer,
		"instance/list":                executeInstanceList,
		"instance/settings":            executeSettings,
		"instance/uninstall":           executeInstanceUninstall,
		"instance/v2":                  executeInstanceV2Legacy,
		"instance/default":             executeDefaultInstance,
		"issue/assign":                 executeAssign,
		"issue/transition":             executeTransition,
		"issue/unassign":               executeUnassign,
		"issue/view":                   executeView,
		"settings":                     executeSettings,
		"subscribe/list":               executeSubscribeList,
		"transition":                   executeTransition,
		"unassign":                     executeUnassign,
		"uninstall":                    executeInstanceUninstall,
		"view":                         executeView,
		"v2revert":                     executeV2Revert,
		"webhook":                      executeWebhookURL,
		"setup":                        executeSetup,
	},
	defaultHandler: executeJiraDefault,
}

const helpTextHeader = "###### Mattermost Jira Plugin - Slash Command Help\n"

const commonHelpText = "\n" +
	"* `/jira connect [jiraURL]` - Connect your Mattermost account to your Jira account\n" +
	"* `/jira disconnect [jiraURL]` - Disconnect your Mattermost account from your Jira account\n" +
	"* `/jira [issue] assign [issue-key] [assignee]` - Change the assignee of a Jira issue\n" +
	"* `/jira [issue] create [text]` - Create a new Issue with 'text' inserted into the description field\n" +
	"* `/jira [issue] transition [issue-key] [state]` - Change the state of a Jira issue\n" +
	"* `/jira [issue] unassign [issue-key]` - Unassign the Jira issue\n" +
	"* `/jira [issue] view [issue-key]` - View the details of a specific Jira issue\n" +
	"* `/jira help` - Launch the Jira plugin command line help syntax\n" +
	"* `/jira me` - Display information about the current user\n" +
	"* `/jira about` - Display build info\n" +
	"* `/jira instance list` - List installed Jira instances\n" +
	"* `/jira instance settings [setting] [value]` - Update your user settings\n" +
	"  * [setting] can be `notifications`\n" +
	"  * [value] can be `on` or `off`\n" +
	""

const sysAdminHelpText = "\n###### For System Administrators:\n" +
	"Setup Jira plugin\n" +
	"* `/jira setup` - Start Jira plugin setup flow\n" +
	"* `/jira webhook [jiraURL]` - Display the webhook URLs to setup on Jira\n" +
	"Install Jira instances:\n" +
	"* `/jira instance install server [jiraURL]` - Connect Mattermost to a Jira Server or Data Center instance located at <jiraURL>\n" +
	"* `/jira instance install cloud-oauth [jiraURL]` - Connect Mattermost to a Jira Cloud instance using OAuth 2.0 located at <jiraURL>\n" +
	"Uninstall Jira instances:\n" +
	"* `/jira instance uninstall server [jiraURL]` - Disconnect Mattermost from a Jira Server or Data Center instance located at <jiraURL>\n" +
	"* `/jira instance uninstall cloud-oauth [jiraURL]` - Disconnect Mattermost from a Jira Cloud instance using OAuth 2.0 located at <jiraURL>\n" +
	"Manage channel subscriptions:\n" +
	"* `/jira subscribe ` - Configure the Jira notifications sent to this channel\n" +
	"* `/jira subscribe list` - Display all the the subscription rules setup across all the channels and teams on your Mattermost instance\n" +
	"Other:\n" +
	"* `/jira instance alias [URL] [alias-name]` - assign an alias to an instance\n" +
	"* `/jira instance unalias [alias-name]` - remve an alias from an instance\n" +
	"* `/jira instance v2 <jiraURL>` - Set the Jira instance to process \"v2\" webhooks and subscriptions (not prefixed with the instance ID)\n" +
	"* `/jira instance default <jiraURL>` - Set a default instance in case of multiple Jira instances\n" +
	"* `/jira webhook [--instance=<jiraURL>]` -  Show the Mattermost webhook to receive JQL queries\n" +
	"* `/jira v2revert ` - Revert to V2 jira plugin data model\n" +
	""

func (p *Plugin) registerJiraCommand(enableAutocomplete, enableOptInstance bool) error {
	// Optimistically unregister what was registered before
	_ = p.client.SlashCommand.Unregister("", commandTrigger)

	command, err := p.createJiraCommand(enableAutocomplete, enableOptInstance)
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	err = p.client.SlashCommand.Register(command)
	if err != nil {
		return errors.Wrapf(err, "failed to register /%s command", commandTrigger)
	}

	return nil
}

func (p *Plugin) createJiraCommand(enableAutocomplete, enableOptInstance bool) (*model.Command, error) {
	jira := model.NewAutocompleteData(
		commandTrigger, "[issue|instance|help|me|about]", "Connect to and interact with Jira")

	if enableAutocomplete {
		addSubCommands(jira, enableOptInstance)
	}

	iconData, err := command.GetIconData(p.API, "assets/icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              jira.Trigger,
		Description:          "Integration with Jira.",
		DisplayName:          "Jira",
		AutoComplete:         true,
		AutocompleteData:     jira,
		AutoCompleteDesc:     jira.HelpText,
		AutoCompleteHint:     jira.Hint,
		AutocompleteIconData: iconData,
	}, nil
}

func addSubCommands(jira *model.AutocompleteData, optInstance bool) {
	// Top-level common commands
	jira.AddCommand(createViewCommand(optInstance))
	jira.AddCommand(createTransitionCommand(optInstance))
	jira.AddCommand(createAssignCommand(optInstance))
	jira.AddCommand(createUnassignCommand(optInstance))
	jira.AddCommand(createConnectCommand())
	jira.AddCommand(createDisconnectCommand())
	jira.AddCommand(createSettingsCommand(optInstance))

	// Generic commands
	jira.AddCommand(createIssueCommand(optInstance))
	jira.AddCommand(createInstanceCommand(optInstance))

	// Admin commands
	jira.AddCommand(createSubscribeCommand(optInstance))
	jira.AddCommand(createWebhookCommand(optInstance))
	jira.AddCommand(createSetupCommand())

	// Help and info
	jira.AddCommand(model.NewAutocompleteData("help", "", "Display help for `/jira` command"))
	jira.AddCommand(model.NewAutocompleteData("me", "", "Display information about the current user"))
	jira.AddCommand(command.BuildInfoAutocomplete("about"))
}

func createInstanceCommand(optInstance bool) *model.AutocompleteData {
	instance := model.NewAutocompleteData(
		"instance", "[alias|connect|disconnect|settings|unalias|default]", "View and manage installed Jira instances; more commands available to system administrators")
	instance.AddCommand(createAliasCommand())
	instance.AddCommand(createUnAliasCommand())
	instance.AddCommand(createConnectCommand())
	instance.AddCommand(createSettingsCommand(optInstance))
	instance.AddCommand(createDisconnectCommand())
	instance.AddCommand(createDefaultInstanceCommand())

	jiraTypes := []model.AutocompleteListItem{
		{HelpText: "Jira Server or Datacenter", Item: "server"},
		{HelpText: "Jira Cloud OAuth 2.0 (atlassian.net)", Item: "cloud-oauth"},
	}

	install := model.NewAutocompleteData(
		"install", "[server|cloud-oauth] [URL]", "Connect Mattermost to a Jira instance")
	install.AddStaticListArgument("Jira type: server, cloud or cloud-oauth", true, jiraTypes)
	install.AddTextArgument("Jira URL", "Enter the Jira URL, e.g. https://mattermost.atlassian.net", "")
	install.RoleID = model.SystemAdminRoleId

	uninstall := model.NewAutocompleteData(
		"uninstall", "[server|cloud-oauth] [URL]", "Disconnect Mattermost from a Jira instance")
	uninstall.AddStaticListArgument("Jira type: server, cloud or cloud-oauth", true, jiraTypes)
	uninstall.AddDynamicListArgument("Jira instance", makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias), true)
	uninstall.RoleID = model.SystemAdminRoleId

	list := model.NewAutocompleteData(
		"list", "", "List installed Jira instances")
	list.RoleID = model.SystemAdminRoleId

	instance.AddCommand(createConnectCommand())
	instance.AddCommand(createDisconnectCommand())
	instance.AddCommand(list)
	instance.AddCommand(createSettingsCommand(optInstance))
	instance.AddCommand(install)
	instance.AddCommand(uninstall)
	return instance
}

func createIssueCommand(optInstance bool) *model.AutocompleteData {
	issue := model.NewAutocompleteData(
		"issue", "[view|assign|transition]", "View and manage Jira issues")
	issue.AddCommand(createViewCommand(optInstance))
	issue.AddCommand(createTransitionCommand(optInstance))
	issue.AddCommand(createAssignCommand(optInstance))
	issue.AddCommand(createUnassignCommand(optInstance))
	return issue
}

func withFlagInstance(cmd *model.AutocompleteData, optInstance bool, route string) {
	if !optInstance {
		return
	}
	cmd.AddNamedDynamicListArgument("instance", "Jira URL", route, false)
}

func withParamIssueKey(cmd *model.AutocompleteData) {
	// TODO: Implement dynamic autocomplete for Jira issue (search)
	cmd.AddTextArgument("Jira issue key", "", "")
}

func createConnectCommand() *model.AutocompleteData {
	connect := model.NewAutocompleteData(
		"connect", "", "Connect your Mattermost account to your Jira account")
	connect.AddDynamicListArgument("Jira URL", makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias), false)
	return connect
}

func createAliasCommand() *model.AutocompleteData {
	alias := model.NewAutocompleteData(
		"alias", "", "Create an alias to your Jira instance")
	alias.AddDynamicListArgument("Jira URL", makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias), false)
	return alias
}

func createUnAliasCommand() *model.AutocompleteData {
	alias := model.NewAutocompleteData(
		"unalias", "", "Remove an alias from a Jira instance")
	alias.AddDynamicListArgument("Jira URL", makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias), false)
	return alias
}

func createDisconnectCommand() *model.AutocompleteData {
	disconnect := model.NewAutocompleteData(
		"disconnect", "[Jira URL]", "Disconnect your Mattermost account from your Jira account")
	disconnect.AddDynamicListArgument("Jira URL", makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias), false)
	return disconnect
}

func createDefaultInstanceCommand() *model.AutocompleteData {
	defaultInstance := model.NewAutocompleteData(
		"default", "[Jira URL]", "Set a default instance in case of multiple Jira instances")
	defaultInstance.AddDynamicListArgument("Jira URL", makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias), true)
	return defaultInstance
}

func createSettingsCommand(optInstance bool) *model.AutocompleteData {
	settings := model.NewAutocompleteData(
		"settings", "[list|notifications]", "View or update your user settings")

	list := model.NewAutocompleteData(
		"list", "", "View your current settings")
	settings.AddCommand(list)

	notifications := model.NewAutocompleteData(
		"notifications", "[on|off]", "Update your user notifications settings")
	notifications.AddStaticListArgument("value", true, []model.AutocompleteListItem{
		{HelpText: "Turn notifications on", Item: "on"},
		{HelpText: "Turn notifications off", Item: "off"},
	})
	withFlagInstance(notifications, optInstance, makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias))
	settings.AddCommand(notifications)

	return settings
}

func createViewCommand(optInstance bool) *model.AutocompleteData {
	view := model.NewAutocompleteData(
		"view", "[issue]", "Display a Jira issue")
	withParamIssueKey(view)
	withFlagInstance(view, optInstance, makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias))
	return view
}

func createTransitionCommand(optInstance bool) *model.AutocompleteData {
	transition := model.NewAutocompleteData(
		"transition", "[Jira issue] [To state]", "Change the state of a Jira issue")
	withParamIssueKey(transition)
	// TODO: Implement dynamic transition autocomplete
	transition.AddTextArgument("To state", "", "")
	withFlagInstance(transition, optInstance, makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias))
	return transition
}

func createAssignCommand(optInstance bool) *model.AutocompleteData {
	assign := model.NewAutocompleteData(
		"assign", "[Jira issue] [user]", "Change the assignee of a Jira issue")
	withParamIssueKey(assign)
	// TODO: Implement dynamic Jira user search autocomplete
	assign.AddTextArgument("User", "", "")
	withFlagInstance(assign, optInstance, makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias))
	return assign
}

func createUnassignCommand(optInstance bool) *model.AutocompleteData {
	unassign := model.NewAutocompleteData(
		"unassign", "[Jira issue]", "Unassign a Jira issue")
	withParamIssueKey(unassign)
	withFlagInstance(unassign, optInstance, makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias))
	return unassign
}

func createSubscribeCommand(optInstance bool) *model.AutocompleteData {
	subscribe := model.NewAutocompleteData(
		"subscribe", "[edit|list]", "List or configure the Jira notifications sent to this channel")
	subscribe.AddCommand(model.NewAutocompleteData(
		"edit", "", "Configure the Jira notifications sent to this channel"))

	list := model.NewAutocompleteData(
		"list", "", "List the Jira notifications sent to this channel")
	withFlagInstance(list, optInstance, makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias))
	subscribe.AddCommand(list)
	return subscribe
}

func createWebhookCommand(optInstance bool) *model.AutocompleteData {
	webhook := model.NewAutocompleteData(
		"webhook", "[Jira URL]", "Display the webhook URLs to set up on Jira")
	webhook.RoleID = model.SystemAdminRoleId
	withFlagInstance(webhook, optInstance, makeAutocompleteRoute(routeAutocompleteInstalledInstanceWithAlias))
	return webhook
}

func createSetupCommand() *model.AutocompleteData {
	setup := model.NewAutocompleteData(
		"setup", "", "Start Jira plugin setup flow")
	setup.RoleID = model.SystemAdminRoleId
	return setup
}

type CommandHandlerFunc func(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse

type CommandHandler struct {
	handlers       map[string]CommandHandlerFunc
	defaultHandler CommandHandlerFunc
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

func executeHelp(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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
	if len(args) > 1 {
		return p.help(header)
	}
	jiraURL := ""
	if len(args) > 0 {
		jiraURL = args[0]
	}
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return p.responsef(header, "Failed to load instances. Error: %v.", err)
	}
	instance := instances.getByAlias(jiraURL)
	if instance != nil {
		jiraURL = instance.InstanceID.String()
	}
	disconnected, err := p.DisconnectUser(jiraURL, types.ID(header.UserId))
	if errors.Cause(err) == kvstore.ErrNotFound {
		errorStr := "Your account is not connected to Jira. Please use `/jira connect` to connect your account."
		if jiraURL != "" {
			errorStr = fmt.Sprintf("You do not currently have a Jira account at %s linked to your Mattermost account. Please use `/jira connect` to connect your account.", jiraURL)
		}
		return p.responsef(header, errorStr)
	}
	if err != nil {
		return p.responsef(header, "Could not complete the **disconnection** request. Error: %v", err)
	}
	return p.responsef(header, "You have successfully disconnected your Jira account (**%s**).", disconnected.DisplayName)
}

func executeDefaultInstance(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) < 1 {
		return p.responsef(header, "Please specify the Jira instance URL")
	}

	if len(args) > 1 {
		return p.help(header)
	}

	jiraURL := ""
	if len(args) > 0 {
		jiraURL = args[0]
	}

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return p.responsef(header, "Failed to load instances. Error: %v.", err)
	}

	instance := instances.getByAlias(jiraURL)
	if instance != nil {
		jiraURL = instance.InstanceID.String()
	}

	if err := p.SetDefaultInstance(jiraURL, types.ID(header.UserId)); err != nil {
		return p.responsef(header, "Could not complete the **default instance** request. Error: %v", err)
	}

	return p.responsef(header, "You have successfully set a default instance for your Jira account (**%s**).", jiraURL)
}

func executeConnect(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	if len(args) > 1 {
		return p.help(header)
	}
	jiraURL := ""
	if len(args) > 0 {
		jiraURL = args[0]
	}
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return p.responsef(header, "Failed to load instances. Error: %v.", err)
	}
	instance := instances.getByAlias(jiraURL)
	if instance != nil {
		jiraURL = instance.InstanceID.String()
	}

	info, err := p.GetUserInfo(types.ID(header.UserId), nil)
	if err != nil {
		return p.responsef(header, "Failed to connect: "+err.Error())
	}
	if info.Instances.IsEmpty() {
		return p.responsef(header,
			"No Jira instances have been installed. Please contact the system administrator.")
	}
	if jiraURL == "" {
		if info.connectable.Len() == 1 {
			jiraURL = info.connectable.IDs()[0].String()
		}
	}
	instanceID := types.ID(jiraURL)
	if info.connectable.IsEmpty() {
		return p.responsef(header,
			"You already have connected all available Jira accounts. Please use `/jira disconnect --instance=%s` to disconnect.",
			instanceID)
	}
	if !info.connectable.Contains(instanceID) {
		return p.responsef(header,
			"Jira instance %s is not installed, please contact the system administrator.",
			instanceID)
	}
	conn, err := p.userStore.LoadConnection(instanceID, types.ID(header.UserId))
	if err == nil && len(conn.JiraAccountID()) != 0 {
		return p.responsef(header,
			"You already have a Jira account linked to your Mattermost account from %s. Please use `/jira disconnect --instance=%s` to disconnect.",
			instanceID, instanceID)
	}

	link := routeUserConnect
	link = instancePath(link, instanceID)
	return p.responsef(header, "[Click here to link your Jira account](%s%s)",
		p.GetPluginURL(), link)
}

func executeInstanceAlias(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira instance alias` can only be run by a system administrator.")
	}

	if len(args) < 2 {
		return p.responsef(header, "Please specify both an instance and alias")
	}

	instanceID := types.ID(args[0])
	alias := strings.Join(args[1:], " ")

	if len(args) > 2 {
		return p.responsef(header, "Alias `%v` is an invalid alias. Please choose an alias without spaces.", alias)
	}

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return p.responsef(header, "Failed to load instances. Error: %v.", err)
	}

	instanceFound := instances.getByAlias(string(instanceID))
	if instanceFound != nil {
		instanceID = instanceFound.InstanceID
	}

	isUnique, id := instances.isAliasUnique(instanceID, alias)
	if !isUnique {
		return p.responsef(header, "Alias `%v` already exists on InstanceID: %v.", alias, id)
	}

	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return p.responsef(header, "Failed to load instance. Error: %v.", err)
	}
	if instance == nil {
		return p.responsef(header, "Failed to get instance. InstanceID: %v.", instanceID)
	}
	instance.Common().Alias = alias

	instances.Set(instance.Common())
	err = p.instanceStore.StoreInstances(instances)
	if err != nil {
		return p.responsef(header, "Failed to save instance. Error: %v.", err)
	}

	instance.Common().Alias = alias
	err = p.instanceStore.StoreInstance(instance)
	if err != nil {
		return p.responsef(header, "Failed to save instance. Error: %v.", err)
	}

	return p.responsef(header, "You have successfully aliased instance %v to `%v`.", instanceID, alias)
}

func executeInstanceUnalias(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira instance unalias` can only be run by a system administrator.")
	}

	if len(args) < 1 {
		return p.responsef(header, "Please specify an alias")
	}

	alias := strings.Join(args, " ")

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return p.responsef(header, "Failed to load instances. Error: %v.", err)
	}

	instanceFound := instances.getByAlias(alias)
	if instanceFound == nil {
		return p.responsef(header, "Instance with alias `%v` does not exist.", alias)
	}

	idFound := instanceFound.InstanceID
	instance, err := p.instanceStore.LoadInstance(idFound)
	if err != nil {
		return p.responsef(header, "Failed to load instance. Error: %v.", err)
	}
	if instance == nil {
		return p.responsef(header, "Failed to get instance. InstanceID: %v.", idFound)
	}
	instance.Common().Alias = ""

	instances.Set(instance.Common())
	err = p.instanceStore.StoreInstances(instances)
	if err != nil {
		return p.responsef(header, "Failed to save instance. Error: %v.", err)
	}

	err = p.instanceStore.StoreInstance(instance)
	if err != nil {
		return p.responsef(header, "Failed to save instance. Error: %v.", err)
	}

	return p.responsef(header, "You have successfully unaliased instance %v from `%v`.", idFound, alias)
}

func executeSettings(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	user, instance, args, err := p.loadFlagUserInstance(header.UserId, args)
	if err != nil {
		return p.responsef(header, "Failed to load your connection to Jira. Error: %v.", err)
	}

	conn, err := p.userStore.LoadConnection(instance.GetID(), user.MattermostUserID)
	if err != nil {
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`. Error: %v.", err)
	}

	if len(args) == 0 {
		return p.responsef(header, "Current settings:\n%s", conn.Settings.String())
	}

	switch args[0] {
	case "list":
		return p.responsef(header, "Current settings:\n%s", conn.Settings.String())
	case "notifications":
		return p.settingsNotifications(header, instance.GetID(), user.MattermostUserID, conn, args)
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
	user, instance, args, err := p.loadFlagUserInstance(header.UserId, args)
	if err != nil {
		return p.responsef(header, "Failed to load your connection to Jira. Error: %v.", err)
	}
	if len(args) != 1 {
		return p.responsef(header, "Please specify an issue key in the form `/jira view <issue-key>`.")
	}

	issueID := args[0]

	conn, err := p.userStore.LoadConnection(instance.GetID(), user.MattermostUserID)
	if err != nil {
		// TODO: try to retrieve the issue anonymously
		return p.responsef(header, "Your username is not connected to Jira. Please type `jira connect`.")
	}

	attachment, err := p.getIssueAsSlackAttachment(instance, conn, strings.ToUpper(issueID), true)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	post := &model.Post{
		UserId:    p.getUserID(),
		ChannelId: header.ChannelId,
		RootId:    header.RootId,
	}
	post.AddProp("attachments", attachment)

	p.client.Post.SendEphemeralPost(header.UserId, post)

	return &model.CommandResponse{}
}

// executeV2Revert reverts the store from v3 to v2 and instructs the user how
// to proceed with downgrading
func executeV2Revert(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira v2revert` can only be run by a system administrator.")
	}

	preMessage := `#### |/jira v2revert| will revert the V3 Jira plugin database to V2. Please use the |--force| flag to complete this command.` + "\n"
	if len(args) == 1 && args[0] == "--force" {
		msg := MigrateV3ToV2(p)
		if msg != "" {
			return p.responsef(header, msg)
		}
		preMessage = `#### Successfully reverted the V3 Jira plugin database to V2. The Jira plugin has been disabled.` + "\n"

		go func() {
			_ = p.client.Plugin.Disable(manifest.Id)
		}()
	}
	message := `**Please note that if you have multiple configured Jira instances this command will result in all non-legacy instances being removed.**

After successfully reverting, please **choose one** of the following:

##### 1. Install Jira plugin |v2.4.0|
Downgrade to install the V2 compatible Jira plugin and use the reverted V2 data models created by the |v2revert| command. The Jira plugin |v2.4.0| can be found via the marketplace or GitHub releases page.

##### 2. Continue using the |v3| data model of the plugin
If you ran |v2revert| unintentionally and would like to continue using the current version of the plugin (|v3+|) you can re-enable the plugin through |System Console| > |PLUGINS| > |Plugin Management|.  This will perform the necessary migration steps to use a |v3+| version of the Jira plugin.`

	message = preMessage + message
	message = strings.ReplaceAll(message, "|", "`")

	p.TrackUserEvent("v2RevertSubmitted", header.UserId, nil)

	return p.responsef(header, message)
}

func executeInstanceList(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira instance list` can only be run by a system administrator.")
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
	text := "| |Alias|URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		instanceID := types.ID(key)
		instanceCommon := instances.Get(instanceID)
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
		format := "|%v|%s|%s|%s|\n"
		if instances.Get(instanceID).IsV2Legacy {
			format = "|%v|%s (v2 legacy)|%s|%s|\n"
		}
		alias := instanceCommon.Alias
		if alias == "" {
			alias = "n/a"
		}
		text += fmt.Sprintf(format, i+1, alias, key, details)
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

	_, instance, args, err := p.loadFlagUserInstance(header.UserId, args)
	if err != nil {
		return p.responsef(header, "Failed to identify the Jira instance. Error: %v.", err)
	}
	if len(args) != 0 {
		return p.responsef(header, "No arguments were expected.")
	}

	msg, err := p.listChannelSubscriptions(instance.GetID(), header.TeamId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

func authorizedSysAdmin(p *Plugin, userID string) (bool, error) {
	user, err := p.client.User.Get(userID)
	if err != nil {
		return false, err
	}
	if !strings.Contains(user.Roles, "system_admin") {
		return false, nil
	}
	return true, nil
}

func executeInstanceInstallCloud(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	if !authorized {
		return p.responsef(header, "`/jira install` can only be run by a system administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}

	jiraURL, err := p.installInactiveCloudInstance(args[0], header.UserId)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	return p.respondCommandTemplate(header, "/command/install_cloud.md", map[string]string{
		"JiraURL":                 jiraURL,
		"PluginURL":               p.GetPluginURL(),
		"AtlassianConnectJSONURL": p.GetPluginURL() + instancePath(routeACJSON, types.ID(jiraURL)),
	})
}

func executeInstanceInstallCloudOAuth(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	if !authorized {
		return p.responsef(header, "`/jira install` can only be run by a Mattermost system administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}

	jiraURL, instance, err := p.installCloudOAuthInstance(args[0])
	if err != nil {
		return p.responsef(header, err.Error())
	}

	state := flow.State{
		keyEdition:          string(CloudOAuthInstanceType),
		keyJiraURL:          jiraURL,
		keyInstance:         instance,
		keyOAuthCompleteURL: p.GetPluginURL() + instancePath(routeOAuth2Complete, types.ID(jiraURL)),
		keyConnectURL:       p.GetPluginURL() + instancePath(routeUserConnect, types.ID(jiraURL)),
	}

	if err = p.oauth2Flow.ForUser(header.UserId).Start(state); err != nil {
		return p.responsef(header, err.Error())
	}

	channel, err := p.client.Channel.GetDirect(header.UserId, p.conf.botUserID)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	if channel != nil && channel.Id != header.ChannelId {
		return p.responsef(header, "continue in the direct conversation with @jira bot.")
	}

	return &model.CommandResponse{}
}

func executeInstanceInstallServer(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	if !authorized {
		return p.responsef(header, "`/jira install` can only be run by a system administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}
	jiraURL, instance, err := p.installServerInstance(args[0])
	if err != nil {
		return p.responsef(header, err.Error())
	}
	pkey, err := p.publicKeyString()
	if err != nil {
		return p.responsef(header, "Failed to load public key: %v", err)
	}

	return p.respondCommandTemplate(header, "/command/install_server.md", map[string]string{
		"JiraURL":       jiraURL,
		"PluginURL":     p.GetPluginURL(),
		"MattermostKey": instance.GetMattermostKey(),
		"PublicKey":     pkey,
	})
}

// executeUninstall will uninstall the jira instance if the url matches, and then update all connected clients
// so that their Jira-related menu options are removed.
func executeInstanceUninstall(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	if !authorized {
		return p.responsef(header, "`/jira uninstall` can only be run by a System Administrator.")
	}
	if len(args) != 2 {
		return p.help(header)
	}

	instanceType := InstanceType(args[0])
	instanceURL := args[1]

	id, err := utils.NormalizeJiraURL(instanceURL)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	uninstalled, err := p.UninstallInstance(types.ID(id), instanceType)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	uninstallInstructions := `` +
		`Jira instance successfully uninstalled. Navigate to [**your app management URL**](%s) in order to remove the application from your Jira instance.
Don't forget to remove Jira-side webhook in [Jira System Settings/Webhooks](%s)'
`
	return p.responsef(header, uninstallInstructions, uninstalled.GetManageAppsURL(), uninstalled.GetManageWebhooksURL())
}

func executeUnassign(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	_, instance, args, err := p.loadFlagUserInstance(header.UserId, args)
	if err != nil {
		return p.responsef(header, "Failed to load your connection to Jira. Error: %v.", err)
	}

	if len(args) != 1 {
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
	_, instance, args, err := p.loadFlagUserInstance(header.UserId, args)
	if err != nil {
		return p.responsef(header, "Failed to load your connection to Jira. Error: %v.", err)
	}

	if len(args) != 2 {
		return p.responsef(header, "Please specify an issue key and an assignee search string, in the form `/jira assign <issue-key> <assignee>`.")
	}
	issueKey := strings.ToUpper(args[0])
	userSearch := strings.Join(args[1:], " ")
	var assignee *jira.User
	if strings.HasPrefix(userSearch, "@") {
		assignee, err = p.GetJiraUserFromMentions(instance.GetID(), header.UserMentions, userSearch)
		if err != nil {
			return p.responsef(header, "%v", err)
		}
	}

	msg, err := p.AssignIssue(instance, types.ID(header.UserId), issueKey, userSearch, assignee)
	if err != nil {
		return p.responsef(header, "%v", err)
	}

	return p.responsef(header, msg)
}

// TODO should transition command post to channel? Options?
func executeTransition(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	instanceURL, args, err := p.parseCommandFlagInstanceURL(args)
	if err != nil {
		return p.responsef(header, "Failed to load your connection to Jira. Error: %v.", err)
	}
	if len(args) < 2 {
		return p.help(header)
	}
	issueKey := strings.ToUpper(args[0])
	toState := strings.Join(args[1:], " ")
	mattermostUserID := types.ID(header.UserId)

	_, instanceID, err := p.ResolveUserInstanceURL(mattermostUserID, instanceURL)
	if err != nil {
		return p.responsef(header, "Failed to identify Jira instance %s. Error: %v.", instanceURL, err)
	}

	msg, err := p.TransitionIssue(&InTransitionIssue{
		InstanceID:       instanceID,
		mattermostUserID: mattermostUserID,
		IssueKey:         issueKey,
		ToState:          toState,
	})
	if err != nil {
		return p.responsef(header, err.Error())
	}

	return p.responsef(header, msg)
}

func executeMe(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
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
	connectionBullet := func(ic *InstanceCommon, connection *Connection, isDefault bool) string {
		id := ic.InstanceID.String()
		if isDefault {
			id = "**" + id + "**"
		}
		switch ic.Type {
		case CloudInstanceType:
			return sbullet(id, fmt.Sprintf("Cloud, connected as **%s** (AccountID: `%s`)",
				connection.User.DisplayName,
				connection.User.AccountID))
		case ServerInstanceType:
			return sbullet(id, fmt.Sprintf("Server, connected as **%s** (Name:%s, Key:%s, EmailAddress:%s)",
				connection.User.DisplayName,
				connection.User.Name,
				connection.User.Key,
				connection.User.EmailAddress))
		}
		return ""
	}

	info, err := p.GetUserInfo(mattermostUserID, nil)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	resp := sbullet("Mattermost site URL", p.GetSiteURL())
	resp += sbullet("Mattermost user ID", fmt.Sprintf("`%s`", mattermostUserID))

	switch {
	case info.IsConnected:
		resp += fmt.Sprintf("###### Connected to %v Jira instances:\n", info.User.ConnectedInstances.Len())
	case info.Instances.Len() > 0:
		resp += "Jira is installed, but you are not connected. Please type `/jira connect` to connect.\n"
	default:
		return p.responsef(header, resp+"\nNo Jira instances installed, please contact your system administrator.")
	}

	if info.IsConnected {
		for _, instanceID := range info.User.ConnectedInstances.IDs() {
			connection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
			if err != nil {
				return p.responsef(header, err.Error())
			}

			resp += connectionBullet(info.User.ConnectedInstances.Get(instanceID), connection, info.User.DefaultInstanceID == instanceID)
			resp += fmt.Sprintf("   * %s\n", connection.Settings)
			if connection.SavedFieldValues != nil && connection.SavedFieldValues.ProjectKey != "" {
				resp += fmt.Sprintf("   * Default project: `%s`\n", connection.SavedFieldValues.ProjectKey)
			}
		}
	}

	orphans := ""
	if !info.Instances.IsEmpty() {
		resp += "\n###### Available Jira instances:\n"
		for _, instanceID := range info.Instances.IDs() {
			encoded := url.PathEscape(encode([]byte(instanceID)))
			ic := info.Instances.Get(instanceID)
			if ic.IsV2Legacy {
				resp += sbullet(instanceID.String(), fmt.Sprintf("%s, **v2 legacy** (`%s`)", ic.Type, encoded))
			} else {
				resp += sbullet(instanceID.String(), fmt.Sprintf("%s (`%s`)", ic.Type, encoded))
			}
		}

		for _, instanceID := range info.Instances.IDs() {
			if info.IsConnected && info.User.ConnectedInstances.Contains(instanceID) {
				continue
			}
			connection, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
			if err != nil {
				if errors.Cause(err) == kvstore.ErrNotFound {
					continue
				}
				return p.responsef(header, err.Error())
			}

			orphans += connectionBullet(info.Instances.Get(instanceID), connection, false)
		}
	}
	if orphans != "" {
		resp += fmt.Sprintf("###### Orphant Jira connections:\n%s", orphans)
	}

	return p.responsef(header, resp)
}

func executeAbout(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	text, err := command.BuildInfo(model.Manifest{
		Id:      manifest.Id,
		Version: manifest.Version,
		Name:    manifest.Name,
	})
	if err != nil {
		text = errors.Wrap(err, "failed to get build info").Error()
	}

	return p.responsef(header, text)
}

func executeWebhookURL(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira webhook` can only be run by a system administrator.")
	}
	jiraURL, args, err := p.parseCommandFlagInstanceURL(args)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if len(args) > 0 {
		return p.help(header)
	}

	instanceID, err := p.ResolveWebhookInstanceURL(jiraURL)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return p.responsef(header, err.Error())
	}

	subWebhookURL, legacyWebhookURL, err := p.GetWebhookURL(jiraURL, header.TeamId, header.ChannelId)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	return p.responsef(header,
		"To set up webhook for instance %s please navigate to [Jira System Settings/Webhooks](%s) where you can add webhooks.\n"+
			"Use `/jira webhook jiraURL` to specify another Jira instance. Use `/jira instance list` to view the available instances.\n"+
			"##### Subscriptions webhook.\n"+
			"Subscriptions webhook needs to be set up once, is shared by all channels and subscription filters.\n"+
			"   - `%s`\n"+
			"   - right-click on [link](%s) and \"Copy Link Address\" to Copy\n"+
			"##### Legacy webhooks\n"+
			"If your organization's infrastructure is set up such that your Mattermost instance cannot connect to your Jira instance, you will not be able to use the Channel Subscriptions feature. You will instead need to use the \"Legacy Webhooks\" feature supported by the Jira plugin.\n"+
			"Legacy webhook needs to be set up for each channel. For this channel:\n"+
			"   - `%s`\n"+
			"   - right-click on [link](%s) and \"Copy Link Address\" to copy\n"+
			" Visit the [Legacy Webhooks](https://mattermost.gitbook.io/plugin-jira/administrator-guide/notification-management#legacy-webhooks) page to learn more about this feature.\n"+
			"",
		instanceID, instance.GetManageWebhooksURL(), subWebhookURL, subWebhookURL, legacyWebhookURL, legacyWebhookURL)
}

func executeSetup(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira setup` can only be run by a system administrator.")
	}

	if err = p.setupFlow.ForUser(header.UserId).Start(nil); err != nil {
		return p.responsef(header, errors.Wrap(err, "Failed to start setup wizard").Error())
	}

	channel, err := p.client.Channel.GetDirect(header.UserId, p.conf.botUserID)
	if err != nil {
		return p.responsef(header, err.Error())
	}
	if channel != nil && channel.Id != header.ChannelId {
		return p.responsef(header, "continue in the direct conversation with @jira bot.")
	}

	return &model.CommandResponse{}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.getUserID(),
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Message:   text,
	}
	p.client.Post.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) responsef(commandArgs *model.CommandArgs, format string, args ...interface{}) *model.CommandResponse {
	p.postCommandResponse(commandArgs, fmt.Sprintf(format, args...))
	return &model.CommandResponse{}
}

func executeInstanceV2Legacy(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	authorized, err := authorizedSysAdmin(p, header.UserId)
	if err != nil {
		return p.responsef(header, "%v", err)
	}
	if !authorized {
		return p.responsef(header, "`/jira instance default` can only be run by a system administrator.")
	}
	if len(args) != 1 {
		return p.help(header)
	}
	instanceID := types.ID(args[0])

	err = p.StoreV2LegacyInstance(instanceID)
	if err != nil {
		return p.responsef(header, "Failed to set default Jira instance %s: %v", instanceID, err)
	}

	return p.responsef(header, "%s is set as the default Jira instance", instanceID)
}

func (p *Plugin) parseCommandFlagInstanceURL(args []string) (string, []string, error) {
	instanceURL := ""
	remaining := []string{}
	afterFlagInstance := false
	for _, arg := range args {
		if afterFlagInstance {
			instanceURL = arg
			afterFlagInstance = false
			continue
		}
		if !strings.HasPrefix(arg, "--instance") {
			remaining = append(remaining, arg)
			continue
		}
		if instanceURL != "" {
			return "", nil, errors.New("--instance may not be specified multiple times")
		}
		str := arg[len("--instance"):]

		// --instance=X
		if strings.HasPrefix(str, "=") {
			instanceURL = str[1:]
			continue
		}

		// --instanceXXX error
		if str != "" {
			return "", nil, errors.Errorf("`%s` is not valid", arg)
		}

		// --instance X
		afterFlagInstance = true
	}
	if afterFlagInstance && instanceURL == "" {
		return "", nil, errors.New("--instance requires a value")
	}

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return "", nil, err
	}

	instance := instances.getByAlias(instanceURL)
	if instance != nil {
		instanceID := instance.Common().InstanceID
		return string(instanceID), remaining, nil
	}

	return instanceURL, remaining, nil
}

func (p *Plugin) loadFlagUserInstance(mattermostUserID string, args []string) (*User, Instance, []string, error) {
	instanceURL, args, err := p.parseCommandFlagInstanceURL(args)
	if err != nil {
		return nil, nil, nil, err
	}

	user, instance, err := p.LoadUserInstance(types.ID(mattermostUserID), instanceURL)
	if err != nil {
		return nil, nil, nil, err
	}
	return user, instance, args, nil
}

func (p *Plugin) respondCommandTemplate(commandArgs *model.CommandArgs, path string, values interface{}) *model.CommandResponse {
	t := p.textTemplates[path]
	if t == nil {
		return p.responsef(commandArgs, "no template found for "+path)
	}
	bb := &bytes.Buffer{}
	err := t.Execute(bb, values)
	if err != nil {
		p.responsef(commandArgs, "failed to format results: %v", err)
	}
	return p.responsef(commandArgs, bb.String())
}
