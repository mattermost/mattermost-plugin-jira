package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	stepWelcome                  flow.Name = "welcome"
	stepDelegate                 flow.Name = "delegate"
	stepDelegateComplete         flow.Name = "delegate-complete"
	stepDelegated                flow.Name = "delegated"
	stepChooseEdition            flow.Name = "choose-edition"
	stepCloudOAuthConfigure      flow.Name = "cloud-oauth-configure"
	stepInstalledJiraApp         flow.Name = "installed-app"
	stepServerAddAppLink         flow.Name = "server-add-link"
	stepServerConfirmAppLink     flow.Name = "server-confirm-link"
	stepServerConfigureAppLink1  flow.Name = "server-configure-link1"
	stepServerConfigureAppLink2  flow.Name = "server-configure-link2"
	stepConnect                  flow.Name = "connect"
	stepConnected                flow.Name = "connected"
	stepWebhook                  flow.Name = "webhook"
	stepWebhookDone              flow.Name = "webhook-done"
	stepAnnouncementQuestion     flow.Name = "announcement-question"
	stepAnnouncementConfirmation flow.Name = "announcement-confirmation"
	stepCancel                   flow.Name = "cancel"
	stepDone                     flow.Name = "done"
)

const (
	keyAtlassianConnectURL = "ACURL"
	keyConnectURL          = "ConnectURL"
	keyDelegatedFromUserID = "DelegatedFromUserID"
	keyDelegatedTo         = "Delegated"
	keyEdition             = "Edition"
	keyJiraURL             = "JiraURL"
	keyInstance            = "Instance"
	keyManageWebhooksURL   = "ManageWebhooksURL"
	keyMattermostKey       = "MattermostKey"
	keyPluginURL           = "PluginURL"
	keyPublicKey           = "PublicKey"
	keyWebhookURL          = "WebhookURL"
	keyOAuthCompleteURL    = "OAuthCompleteURL"
)

func (p *Plugin) NewSetupFlow() *flow.Flow {
	pluginURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + Manifest.Id
	conf := p.getConfig()
	return flow.NewFlow("setup-wizard", p.client, pluginURL, conf.botUserID).
		WithSteps(
			p.stepWelcome(),
			p.stepDelegate(),
			p.stepDelegated(),
			p.stepDelegateComplete(),
			p.stepChooseEdition(),

			// Jira Cloud OAuth steps
			p.stepCloudOAuthConfigure(),

			// Jira server steps
			p.stepServerAddAppLink(),
			p.stepServerConfirmAppLink(),
			p.stepServerConfigureAppLink1(),
			p.stepServerConfigureAppLink2(),

			p.stepInstalledJiraApp(),
			p.stepWebhook(),
			p.stepWebhookDone(),
			p.stepConnect(),
			p.stepConnected(),
			p.stepAnnouncementQuestion(),
			p.stepAnnouncementConfirmation(),
			p.stepCancel(),
			p.stepDone(),
		).
		// WithDebugLog().
		InitHTTP(p.router)
}

func (p *Plugin) NewOAuth2Flow() *flow.Flow {
	pluginURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + Manifest.Id
	conf := p.getConfig()
	return flow.NewFlow("setup-oauth2", p.client, pluginURL, conf.botUserID).
		WithSteps(
			p.stepCloudOAuthConfigure(),
			p.stepInstalledJiraApp(),
			p.stepWebhook(),
			p.stepWebhookDone(),
			p.stepConnect(),
			p.stepConnected(),
			p.stepAnnouncementQuestion(),
			p.stepAnnouncementConfirmation(),
			p.stepCancel(),
			p.stepDone(),
		).
		InitHTTP(p.router)
}

var cancelButton = flow.Button{
	Name:    "Cancel setup",
	Color:   flow.ColorDanger,
	OnClick: flow.Goto(stepCancel),
}

func continueButton(next flow.Name) flow.Button {
	return flow.Button{
		Name:    "Continue",
		Color:   flow.ColorPrimary,
		OnClick: flow.Goto(next),
	}
}

func (p *Plugin) stepWelcome() flow.Step {
	return flow.NewStep(stepWelcome).
		WithPretext(":wave: Welcome to Jira integration! [Learn more](https://github.com/mattermost/mattermost-plugin-jira#readme)").
		WithText("Just a few more configuration steps to go!\n" +
			"1. Choose your Jira edition.\n" +
			"2. Create an incoming application link.\n" +
			"3. Configure a Jira subscription webhook.\n" +
			"4. Connect your user account.\n" +
			"\n" +
			"You can **Cancel** setup at any time, and use `/jira` command to complete the configuration later. " +
			"See the [documentation](https://mattermost.gitbook.io/plugin-jira/setting-up/configuration) for details.").
		OnRender(func(f *flow.Flow) {
			p.trackSetupWizard("setup_wizard_start", map[string]interface{}{
				"from_invite": f.GetState().GetString(keyDelegatedFromUserID) != "",
			})(f)
		}).
		WithButton(continueButton(stepDelegate)).
		WithButton(cancelButton)
}

func (p *Plugin) stepDelegate() flow.Step {
	return flow.NewStep(stepDelegate).
		WithText(
			"Configuring the integration requires administrator access to Jira. Are you setting this Jira integration up yourself, or is someone else?").
		WithButton(flow.Button{
			Name:    "I'll do it myself",
			Color:   flow.ColorPrimary,
			OnClick: flow.Goto(stepChooseEdition),
		}).
		WithButton(flow.Button{
			Name:  "I need someone else",
			Color: flow.ColorDefault,
			Dialog: &model.Dialog{
				Title:       "Send instructions to:",
				SubmitLabel: "Send",
				Elements: []model.DialogElement{
					{
						DisplayName: "Jira Admin",
						Name:        "aider",
						Type:        "select",
						DataSource:  "users",
						Placeholder: "Search for people",
						HelpText:    "A Jira admin who can finish setting up the Mattermost integration in Jira.",
					},
				},
			},
			OnDialogSubmit: p.submitDelegateSelection,
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepDelegated() flow.Step {
	return flow.NewStep(stepDelegated).
		WithText("Asked {{.Delegated}} to finish configuring the integration. They will receive a notification to complete the configuration.").
		WithButton(flow.Button{
			Name:     "Waiting for {{.Delegated}}...",
			Color:    flow.ColorDefault,
			Disabled: true,
		}).
		OnRender(p.trackSetupWizard("setup_wizard_delegated", nil)).
		WithButton(cancelButton)
}

func (p *Plugin) stepDelegateComplete() flow.Step {
	return flow.NewStep(stepDelegateComplete).
		WithText("{{.Delegated}} completed configuring the integration. :tada:").
		OnRender(p.trackSetupWizard("setup_wizard_delegate_complete", nil)).
		Next(stepAnnouncementQuestion)
}

func (p *Plugin) stepChooseEdition() flow.Step {
	return flow.NewStep(stepChooseEdition).
		WithPretext("##### :white_check_mark: Step 1: Which Jira edition do you use?").
		WithTitle("Cloud (OAuth 2.0) or Server (on-premise).").
		WithText("Choose whether you're using Jira Cloud (OAuth 2.0) or Jira Server (on-premise/Data Center) edition. " +
			"To integrate with more than one Jira instance, see the [documentation](https://mattermost.gitbook.io/plugin-jira/)").
		WithButton(
			flow.Button{
				Name:  "Jira Cloud (OAuth 2.0)",
				Color: flow.ColorPrimary,
				Dialog: &model.Dialog{
					Title:            "Enter your Jira Cloud URL",
					IntroductionText: "Enter a Jira Cloud URL (typically, `https://yourorg.atlassian.net`), or just the organization part, `yourorg`",
					SubmitLabel:      "Continue",
					Elements: []model.DialogElement{
						{
							DisplayName: "Jira Cloud organization",
							Name:        "url",
							Type:        "text",
							// text, not URL since normally just the org name needs
							// to be entered.
							SubType: "text",
						},
					},
				},
				OnDialogSubmit: p.initCreateCloudOAuthInstance,
			}).
		WithButton(flow.Button{
			Name:  "Jira Server",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:       "Enter Jira Server URL",
				SubmitLabel: "Continue",
				Elements: []model.DialogElement{
					{
						DisplayName: "Jira Server URL",
						Name:        "url",
						Type:        "text",
						SubType:     "url",
					},
				},
			},
			OnDialogSubmit: p.submitCreateServerInstance,
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerAddAppLink() flow.Step {
	return flow.NewStep(stepServerAddAppLink).
		WithPretext("##### :white_check_mark: Step 2: Configure the Mattermost Application Link in Jira").
		WithTitle("Create an incoming application Link.").
		WithText("Jira server {{.JiraURL}} has been successfully added. " +
			"To finish the configuration, we'll need to add and configure an Application Link in your Jira instance.\n" +
			"Complete the following steps, then come back here to select **Continue**.\n\n" +
			"1. Navigate to [**Settings > Applications > Application Links**]({{.JiraURL}}/plugins/servlet/applinks/listApplicationLinks) (see _screenshot_).\n" +
			"2. Keep checked the Atlassian Product Application Type and enter `{{.PluginURL}}` [link]({{.PluginURL}}) as the application link, then select **Create new link**.").
		WithImage("public/server-create-applink.png").
		OnRender(p.trackSetupWizard("setup_wizard_jira_config_start", map[string]interface{}{
			keyEdition: ServerInstanceType,
		})).
		WithButton(continueButton(stepServerConfirmAppLink)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfirmAppLink() flow.Step {
	return flow.NewStep(stepServerConfirmAppLink).
		WithTitle("Confirm Application Link URL.").
		WithText("Ignore any errors in Jira's **Configure Application URL** confirmation screen (see _screenshot_), then select **Continue**.\n").
		WithImage("public/server-confirm-applink-url.png").
		WithButton(continueButton(stepServerConfigureAppLink1)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfigureAppLink1() flow.Step {
	return flow.NewStep(stepServerConfigureAppLink1).
		WithTitle("Create Incoming Application Link.").
		WithText("Complete the following steps in Jira, then come back here to select **Continue**.\n\n" +
			"1. In Jira's **Link Applications** screen (see _screenshot_) enter the following values, and leave all other fields blank.\n" +
			"  - **Application Name**:  `Mattermost`\n" +
			"  - **Application Type**: **Generic Application**\n" +
			"  - **Create incoming link**: :heavy_check_mark: **(important)**\n" +
			"2. Select **Continue**.\n").
		WithImage("public/server-configure-applink-1.png").
		WithButton(continueButton(stepServerConfigureAppLink2)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfigureAppLink2() flow.Step {
	return flow.NewStep(stepServerConfigureAppLink2).
		WithTitle("Configure Incoming Application Link.").
		WithText("Complete the following steps in Jira, then come back here to select **Continue**.\n\n" +
			"1. In Jira's second **Link Applications** screen (see _screenshot_) enter the following values, and leave all other fields blank.\n" +
			"  - **Consumer Key**: `{{.MattermostKey}}`\n" +
			"  - **Consumer Name**: `Mattermost`\n" +
			"  - **Public Key**:\n```\n{{ .PublicKey }}\n```\n" +
			"2. Select **Continue**.\n").
		WithImage("public/server-configure-applink-2.png").
		WithButton(continueButton(stepInstalledJiraApp)).
		WithButton(cancelButton)
}

func (p *Plugin) stepCloudOAuthConfigure() flow.Step {
	return flow.NewStep(stepCloudOAuthConfigure).
		WithPretext("##### :white_check_mark: Step 2: Register an OAuth 2.0 Application in Jira").
		WithText(fmt.Sprintf("Complete the following steps, then come back here to select **Configure**.\n\n"+
			"1. Create an OAuth 2.0 application in Jira from the [Developer console](https://developer.atlassian.com/console/myapps/create-3lo-app/).\n"+
			"2. Name your app according to its purpose, for example:  `Mattermost Jira Plugin - <your company name>`.\n"+
			"3. Accept the **Terms** and click **Create**.\n"+
			"4. Select **Permissions** in the left menu. Next to the JIRA API, select **Add**.\n"+
			"5. Then select **Configure** and ensure following scopes are selected:\n"+
			"       %s\n"+
			"6. Select **Authorization** in the left menu.\n"+
			"7. Next to OAuth 2.0 (3LO), select **Add** and set the Callback URL as follows and click **Save Changes**:\n"+
			"       {{.OAuthCompleteURL}}\n"+
			"8. Select **Settings** in the left menu.\n"+
			"9. Copy the **Client ID** and **Secret** and keep it handy.\n"+
			"10. Click on the **Configure** button below, enter these details and then **Continue**.", JiraScopes)).
		WithButton(flow.Button{
			Name:  "Configure",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:       "Configure your Jira Cloud OAuth 2.0",
				SubmitLabel: "Continue",
				Elements: []model.DialogElement{
					{
						DisplayName: "Jira Cloud organization",
						Name:        "url",
						Type:        "text",
						Default:     `{{.JiraURL}}`,
						SubType:     "text",
					},
					{
						DisplayName: "Jira OAuth Client ID",
						Name:        "client_id",
						Type:        "text",
						SubType:     "text",
						HelpText:    "The client ID for the OAuth app registered with Jira",
					},
					{
						DisplayName: "Jira OAuth Client Secret",
						Name:        "client_secret",
						Type:        "text",
						SubType:     "text",
						HelpText:    "The client secret for the OAuth app registered with Jira",
					},
				},
			},
			OnDialogSubmit: p.submitCreateCloudOAuthInstance,
		}).
		OnRender(p.trackSetupWizard("setup_wizard_cloud_oauth2_configure", nil)).
		WithButton(cancelButton)
}

func (p *Plugin) stepInstalledJiraApp() flow.Step {
	next := func(to flow.Name) func(*flow.Flow) (flow.Name, flow.State, error) {
		return func(f *flow.Flow) (flow.Name, flow.State, error) {
			jiraURL := f.GetState().GetString(keyJiraURL)
			instanceID := types.ID(jiraURL)
			return to, flow.State{
				keyConnectURL:        p.GetPluginURL() + "/" + instancePath(routeUserConnect, instanceID),
				keyWebhookURL:        p.getSubscriptionsWebhookURL(instanceID),
				keyManageWebhooksURL: cloudManageWebhooksURL(jiraURL),
			}, nil
		}
	}
	return flow.NewStep(stepInstalledJiraApp).
		WithText("You've finished configuring the Mattermost App in Jira. Select **Continue** to set up the subscription webhook " +
			"for sending notifications to Mattermost.").
		OnRender(func(f *flow.Flow) {
			p.trackSetupWizard("setup_wizard_jira_config_complete", map[string]interface{}{
				keyEdition: f.GetState().GetString(keyEdition),
			})(f)
		}).
		WithButton(flow.Button{
			Name:    "Continue",
			Color:   flow.ColorPrimary,
			OnClick: next(stepWebhook),
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepWebhook() flow.Step {
	return flow.NewStep(stepWebhook).
		WithPretext("##### :white_check_mark: Step 3: Setup Jira Subscriptions Webhook").
		WithText(`To receive Jira event notifications in Mattermost Channels, you need to set up a single global ` +
			"webhook, configured for all possible event triggers that you would like to be pushed into " +
			"Mattermost. The plugin processes all data from the global webhook, and then routes the events " +
			"to channels and users based on your subscriptions.\n\n" +
			"1. Navigate to [Jira System Settings/Webhooks]({{.ManageWebhooksURL}}) (see _screenshot_), select **Create a WebHook** in the top right corner.\n" +
			"2. Give your webhook a meaningful **Name** of your choice.\n" +
			"3. **Status**: Enabled.\n" +
			"4. Leave **URL** blank for the moment. Once you are done configuring the webhook options, come back " +
			"here and select **View Webhook URL** to see the confidential URL.\n" +
			"5. **Issue related events**: we recommend leaving the query at **All Issues**. Check **Comment**, " +
			"**Attachment**, and **Issue** events. We recommend checking all of these boxes. These events will be " +
			"further filtered by Mattermost subscriptions. Leave **Entity property**, **Worklog**, and **Issue " +
			"link** events unchecked, they are not yet supported.\n" +
			"6. Leave all other checkboxes blank.\n" +
			"7. Select **View Webhook URL** to see the secret **URL** to enter in Jira, and continue.\n").
		WithImage("public/configure-webhook.png").
		OnRender(p.trackSetupWizard("setup_wizard_webhook_start", nil)).
		WithButton(flow.Button{
			Name:  "View webhook URL",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "Jira Webhook URL",
				IntroductionText: "Please scroll to select the entire URL if necessary.\n\n```{{.WebhookURL}}```\n\nOnce you have entered all options and the webhook URL, select **Create**",
				SubmitLabel:      "Continue",
			},
			OnDialogSubmit: flow.DialogGoto(stepWebhookDone),
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepWebhookDone() flow.Step {
	return flow.NewStep(stepWebhookDone).
		WithTitle("Success! Webhook setup is complete. :tada:").
		WithText("You can now use the command `/jira subscribe` from a specific channel to receive Jira notifications in that channel.").
		OnRender(p.trackSetupWizard("setup_wizard_webhook_complete", nil)).
		Next(stepConnect)
}

func (p *Plugin) stepConnect() flow.Step {
	return flow.NewStep(stepConnect).
		WithPretext("##### :white_check_mark: Step 4: Connect your Jira user account").
		WithText("Go **[here]({{.ConnectURL}})** to connect your account.").
		OnRender(p.trackSetupWizard("setup_wizard_user_connect_start", nil)).
		WithButton(flow.Button{
			Name:     "Waiting for confirmation...",
			Color:    flow.ColorDefault,
			Disabled: true,
		})
}

func (p *Plugin) stepConnected() flow.Step {
	return flow.NewStep(stepConnected).
		WithText("You've successfully connected your Mattermost user account to Jira.").
		OnRender(p.trackSetupWizard("setup_wizard_user_connect_complete", nil)).
		Next(stepAnnouncementQuestion)
}

func (p *Plugin) stepAnnouncementQuestion() flow.Step {
	return flow.NewStep(stepAnnouncementQuestion).
		WithPretext("##### :tada: Success! You've successfully set up your Mattermost Jira integration!").
		WithText("You can now:\n" +
			"- Subscribe channels in Mattermost to receive updates from Jira with `/jira subscribe` command (navigate to the target channel first).\n" +
			"- Create Jira issues from posts in Mattermost by selecting **Create Jira Issue** from the **...** menu of the relevant post.\n" +
			"- Attach Mattermost posts to Jira issues as comments by selecting **Attach to Jira Issue** from the **...** menu.\n" +
			"- Control your personal notifications from Jira with `/jira instance settings` command.\n\n" +
			"Want to let your team know?\n").
		OnRender(p.trackSetupWizard("setup_wizard_announcement_start", nil)).
		WithButton(flow.Button{
			Name:  "Send message",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:       "Notify your team",
				SubmitLabel: "Send message",
				Elements: []model.DialogElement{
					{
						DisplayName: "To",
						Name:        "channel_id",
						Type:        "select",
						Placeholder: "Select channel",
						DataSource:  "channels",
					},
					{
						DisplayName: "Message",
						Name:        "message",
						Type:        "textarea",
						Default: "Hi team,\n\n" +
							"We've added an integration that connects Jira and Mattermost. You can get notified when you are mentioned in Jira comments, " +
							"or quickly change a message in Mattermost into a ticket in Jira. It's easy to get started, run the `/jira connect` slash " +
							"command from any channel within Mattermost to connect your user account. See the " +
							"[documentation](https://mattermost.gitbook.io/plugin-jira/end-user-guide/getting-started) for details on using the Jira plugin.",
						HelpText: "You can edit this message before sending it.",
					},
				},
			},
			OnDialogSubmit: p.submitChannelAnnouncement,
		}).
		WithButton(flow.Button{
			Name:    "Not now",
			Color:   flow.ColorDefault,
			OnClick: flow.Goto(stepDone),
		})
}

func (p *Plugin) submitChannelAnnouncement(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	channelIDRaw, ok := submitted["channel_id"]
	if !ok {
		return "", nil, nil, errors.New("channel_id missing")
	}
	channelID, ok := channelIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("channel_id is not a string")
	}

	channel, err := p.client.Channel.Get(channelID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to get channel")
	}

	messageRaw, ok := submitted["message"]
	if !ok {
		return "", nil, nil, errors.New("message is not a string")
	}
	message, ok := messageRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("message is not a string")
	}

	post := &model.Post{
		UserId:    f.UserID,
		ChannelId: channel.Id,
		Message:   message,
	}
	err = p.client.Post.CreatePost(post)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to create announcement post")
	}

	return stepAnnouncementConfirmation, flow.State{
		"ChannelName": channel.Name,
	}, nil, nil
}

func (p *Plugin) stepAnnouncementConfirmation() flow.Step {
	return flow.NewStep(stepAnnouncementConfirmation).
		WithText("Sent the announcement to ~{{ .ChannelName }}.").
		OnRender(p.trackSetupWizard("setup_wizard_announcement_complete", nil)).
		Next(stepDone)
}

func (p *Plugin) stepCancel() flow.Step {
	return flow.NewStep(stepCancel).
		Terminal().
		WithColor(flow.ColorDanger).
		// WithPretext("##### :no_entry_sign: Canceled").
		WithText("Jira integration set up has been canceled. Run it again later using the `/jira setup` command, " +
			"or refer to the [documentation](https://mattermost.gitbook.io/plugin-jira/setting-up) " +
			"to configure it manually.\n").
		OnRender(p.trackSetupWizard("setup_wizard_canceled", nil))
}

func (p *Plugin) stepDone() flow.Step {
	return flow.NewStep(stepDone).
		Terminal().
		WithText(":wave: All done!").
		OnRender(func(f *flow.Flow) {
			delegatedFrom := f.GetState().GetString(keyDelegatedFromUserID)
			if delegatedFrom != "" {
				_ = p.setupFlow.ForUser(delegatedFrom).Go(stepDelegateComplete)
			}
			p.trackSetupWizard("setup_wizard_complete", nil)
		})
}

func (p *Plugin) submitDelegateSelection(f *flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	aiderIDRaw, ok := submission["aider"]
	if !ok {
		return "", nil, nil, errors.New("aider missing")
	}
	aiderID, ok := aiderIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("aider is not a string")
	}

	aider, err := p.client.User.Get(aiderID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed get user")
	}

	err = p.setupFlow.ForUser(aider.Id).Start(flow.State{
		keyDelegatedFromUserID: f.UserID,
	})
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to start configration wizzard")
	}

	return stepDelegated, flow.State{
		keyDelegatedTo: aider.GetDisplayName(model.ShowNicknameFullName),
	}, nil, nil
}

var jiraOrgRegexp = regexp.MustCompile(`^[\w-]+$`)

func (p *Plugin) initCreateCloudOAuthInstance(f *flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, nil, errors.New("no Jira cloud URL in the request")
	}
	jiraURL = strings.TrimSpace(jiraURL)
	if jiraOrgRegexp.MatchString(jiraURL) {
		jiraURL = fmt.Sprintf("https://%s.atlassian.net", jiraURL)
	}

	jiraURL, instance, err := p.installCloudOAuthInstance(jiraURL, "", "")
	if err != nil {
		return "", nil, nil, err
	}

	return stepCloudOAuthConfigure, flow.State{
		keyEdition:          string(CloudOAuthInstanceType),
		keyJiraURL:          jiraURL,
		keyInstance:         instance,
		keyOAuthCompleteURL: p.GetPluginURL() + instancePath(routeOAuth2Complete, types.ID(jiraURL)),
		keyConnectURL:       p.GetPluginURL() + instancePath(routeUserConnect, types.ID(jiraURL)),
	}, nil, nil
}

func (p *Plugin) submitCreateCloudOAuthInstance(f *flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, nil, errors.New("no Jira cloud URL in the request")
	}
	jiraURL = strings.TrimSpace(jiraURL)
	if jiraOrgRegexp.MatchString(jiraURL) {
		jiraURL = fmt.Sprintf("https://%s.atlassian.net", jiraURL)
	}

	clientID, _ := submission["client_id"].(string)
	if clientID == "" {
		return "", nil, nil, errors.New("no Jira OAuth Client ID in the request")
	}

	clientSecret, _ := submission["client_secret"].(string)
	if clientSecret == "" {
		return "", nil, nil, errors.New("no Jira OAuth Client Secret in the request")
	}

	jiraURL, instance, err := p.installCloudOAuthInstance(jiraURL, clientID, clientSecret)
	if err != nil {
		return "", nil, nil, err
	}

	return stepInstalledJiraApp, flow.State{
		keyEdition:          string(CloudOAuthInstanceType),
		keyJiraURL:          jiraURL,
		keyInstance:         instance,
		keyOAuthCompleteURL: p.GetPluginURL() + instancePath(routeOAuth2Complete, types.ID(jiraURL)),
		keyConnectURL:       p.GetPluginURL() + instancePath(routeUserConnect, types.ID(jiraURL)),
	}, nil, nil
}

func (p *Plugin) submitCreateServerInstance(f *flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, nil, errors.New("no Jira server URL in the request")
	}
	jiraURL = strings.TrimSpace(jiraURL)

	jiraURL, si, err := p.installServerInstance(jiraURL)
	if err != nil {
		return "", nil, nil, err
	}
	pkey, err := p.publicKeyString()
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to load public key")
	}

	return stepServerAddAppLink, flow.State{
		keyEdition:           string(ServerInstanceType),
		keyJiraURL:           jiraURL,
		keyPluginURL:         p.GetPluginURL(),
		keyMattermostKey:     si.GetMattermostKey(),
		keyPublicKey:         pkey,
		keyConnectURL:        p.GetPluginURL() + "/" + instancePath(routeUserConnect, si.InstanceID),
		keyWebhookURL:        p.getSubscriptionsWebhookURL(si.InstanceID),
		keyManageWebhooksURL: si.GetManageWebhooksURL(),
	}, nil, nil
}

func (p *Plugin) trackSetupWizard(event string, args map[string]interface{}) func(f *flow.Flow) {
	return func(f *flow.Flow) {
		p.TrackUserEvent(event, f.UserID, args)
	}
}
