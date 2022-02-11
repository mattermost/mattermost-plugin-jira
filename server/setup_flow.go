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
	stepSetupWelcome             flow.Name = "setup-welcome"
	stepDelegateComplete         flow.Name = "delegate-complete"
	stepDelegated                flow.Name = "delegated"
	stepChooseEdition            flow.Name = "choose-edition"
	stepCloudAddedInstance       flow.Name = "cloud-added"
	stepCloudEnableDeveloperMode flow.Name = "cloud-enable-dev"
	stepCloudUploadApp           flow.Name = "cloud-upload-app"
	stepCloudInstalledApp        flow.Name = "cloud-installed"
	stepServerAddAppLink         flow.Name = "server-add-link"
	stepServerConfirmAppLink     flow.Name = "server-confirm-link"
	stepServerConfigureAppLink1  flow.Name = "server-configure-link1"
	stepServerConfigureAppLink2  flow.Name = "server-configure-link2"
	stepConnect                  flow.Name = "connect"
	stepConnected                flow.Name = "connected"
	stepWebhook                  flow.Name = "webhook"
	stepWebhookDone              flow.Name = "webhook-done"
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
	keyManageWebhooksURL   = "ManageWebhooksURL"
	keyMattermostKey       = "MattermostKey"
	keyPluginURL           = "PluginURL"
	keyPublicKey           = "PublicKey"
	keyWebhookURL          = "WebhookURL"
)

func (p *Plugin) NewSetupFlow() *flow.Flow {
	pluginURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + manifest.ID
	conf := p.getConfig()
	return flow.NewFlow("setup-wizard", p.client, pluginURL, conf.botUserID).
		WithSteps(
			p.stepWelcome(),
			p.stepDelegated(),
			p.stepDelegateComplete(),
			p.stepChooseEdition(),

			// Jira Cloud steps
			p.stepCloudAddedInstance(),
			p.stepCloudEnableDeveloperMode(),
			p.stepCloudUploadApp(),
			p.stepCloudInstalledApp(),

			// Jira server steps
			p.stepServerAddAppLink(),
			p.stepServerConfirmAppLink(),
			p.stepServerConfigureAppLink1(),
			p.stepServerConfigureAppLink2(),

			p.stepWebhook(),
			p.stepWebhookDone(),
			p.stepConnect(),
			p.stepConnected(),
			p.stepCancel(),
			p.stepDone(),
		).
		WithDebugLog().
		InitHTTP(p.gorillaRouter)
}

var cancelButton = flow.Button{
	Name:    "Cancel",
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

func skipButton(next flow.Name) flow.Button {
	return flow.Button{
		Name:    "DEBUG Skip",
		Color:   flow.ColorWarning,
		OnClick: flow.Goto(next),
	}
}

func (p *Plugin) stepWelcome() flow.Step {
	return flow.NewStep(stepSetupWelcome).
		WithPretext("##### :wave: Welcome to Jira integration! [Learn more](https://github.com/mattermost/mattermost-plugin-jira#readme)").
		WithTitle("Configure the integration.").
		WithText("Just a few steps to go!\n" +
			"1. Choose the Jira edition (cloud or server) you will connect to.\n" +
			"2. Configure the Mattermost integration (app) in Jira.\n" +
			"3. Configure the subscriptions webhook in Jira.\n" +
			"4. Connect your user account.\n" +
			"Configuring the integration requires administrator access to Jira. If you aren't a Jira admin, " +
			"select **I need someone else** to ask another Mattermost user to do it.\n" +
			"\n" +
			"You can **Cancel** these steps at any time and use the `/jira` command to finish the configuration later. " +
			"See [documentation](https://mattermost.gitbook.io/plugin-jira/setting-up/configuration) for more.").
		WithButton(continueButton(stepChooseEdition)).
		WithButton(flow.Button{
			Name:  "I need someone else",
			Color: flow.ColorDefault,
			Dialog: &model.Dialog{
				Title:       "Send instructions",
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
		WithText("Asked {{.Delegated}} to finish configuring the integration.").
		WithButton(flow.Button{
			Name:     "Waiting for {{.Delegated}}...",
			Color:    flow.ColorDefault,
			Disabled: true,
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepDelegateComplete() flow.Step {
	return flow.NewStep(stepDelegateComplete).
		WithText("{{.Delegated}} completed configuring the integration.").
		Next(stepConnect)
}

func (p *Plugin) stepChooseEdition() flow.Step {
	return flow.NewStep(stepChooseEdition).
		WithPretext("##### :white_check_mark: Step 1: Choose Jira Edition.").
		WithTitle("Cloud or Server (on-premise).").
		WithText("Please choose whether you use the Jira Cloud or Server (on-premise) edition. " +
			"If you need to integrate with more than one Jira instance, please refer to the [documentation](https://mattermost.gitbook.io/plugin-jira/)").
		WithButton(flow.Button{
			Name:  "Jira Cloud",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "Enter Jira Cloud Organization",
				IntroductionText: "Enter Jira Cloud URL (usually, `https://yourorg.atlassian.net`), or just the organization part, `yourorg`.",
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
			OnDialogSubmit: p.submitCreateCloudInstance,
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
		WithPretext("##### :white_check_mark: Step 2: configure the Mattermost application link in Jira").
		WithTitle("Create Application Link.").
		WithText("Jira server {{.JiraURL}} has been successfully added. "+
			"To finish the configuration add and configure an Application Link in your Jira instance.\n"+
			"Complete the following steps, then come back here and select **Continue**.\n\n"+
			"1. Navigate to [**Settings > Applications > Application Links**]({{.JiraURL}}/plugins/servlet/applinks/listApplicationLinks) (see _screenshot_).\n"+
			"2. Enter `{{.PluginURL}}` [link]({{.PluginURL}})as the application link, then select **Create new link**.").
		WithImage(p.GetPluginURL(), "public/server-create-applink.png").
		WithButton(continueButton(stepServerConfirmAppLink)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfirmAppLink() flow.Step {
	return flow.NewStep(stepServerConfirmAppLink).
		WithTitle("Confirm Application Link URL.").
		WithText("Ignore any errors in Jira's **Configure Application URL** confirmation screen (see _screenshot_), then select **Continue**.\n").
		WithImage(p.GetPluginURL(), "public/server-confirm-applink-url.png").
		WithButton(continueButton(stepServerConfigureAppLink1)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfigureAppLink1() flow.Step {
	return flow.NewStep(stepServerConfigureAppLink1).
		WithTitle("Create Incoming Application Link.").
		WithText("Complete the following steps, then come back here and select **Continue**.\n\n"+
			"1. In Jira's **Link Applications** screen (see _screenshot_) enter the following values, leave all other fields blank.\n"+
			"  - **Application Name**:  `Mattermost`\n"+
			"  - **Application Type**: **Generic Application**\n"+
			"  - **Create incoming link**: :heavy_check_mark: **(important)**\n"+
			"2. Select **Continue**.\n").
		WithImage(p.GetPluginURL(), "public/server-configure-applink-1.png").
		WithButton(continueButton(stepServerConfigureAppLink2)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfigureAppLink2() flow.Step {
	return flow.NewStep(stepServerConfigureAppLink2).
		WithTitle("Configure Incoming Application Link.").
		WithText("Complete the following steps, then come back here and select **Continue**.\n\n"+
			"1. In Jira's second **Link Applications** screen (see _screenshot_) enter the following values, leave all other fields blank.\n"+
			"  - **Consumer Key**: `{{.MattermostKey}}`\n"+
			"  - **Consumer Name**: `Mattermost`\n"+
			"  - **Public Key**:\n```\n{{ .PublicKey }}\n```\n"+
			"2. Select **Continue**.\n").
		WithImage(p.GetPluginURL(), "public/server-configure-applink-2.png").
		WithButton(continueButton(stepWebhook)).
		WithButton(cancelButton)
}

func (p *Plugin) stepCloudAddedInstance() flow.Step {
	return flow.NewStep(stepCloudAddedInstance).
		WithText("Jira cloud {{.JiraURL}} has been added and is ready to configure.").
		Next(stepCloudEnableDeveloperMode)
}

func (p *Plugin) stepCloudEnableDeveloperMode() flow.Step {
	return flow.NewStep(stepCloudEnableDeveloperMode).
		WithPretext("##### :white_check_mark: Step 2: configure the Mattermost app in Jira").
		WithTitle("Enable development mode.").
		WithText("Mattermost Jira Cloud integration requires setting your Jira to _development mode_ (see _screenshot_). "+
			"Enabling the development mode allows you to install apps like Mattermost, from outside the Atlassian Marketplace.\n"+
			"Complete the following steps, then come back here and select **Continue**.\n\n"+
			"1. Navigate to [**Settings > Apps > Manage Apps**]({{.JiraURL}}/plugins/servlet/upm?source=side_nav_manage_addons).\n"+
			"2. Select **Settings** at the bottom of the page.\n"+
			"3. Select **Enable development mode**, then select **Apply**.\n").
		WithImage(p.GetPluginURL(), "public/cloud-enable-dev-mode.png").
		WithButton(continueButton(stepCloudUploadApp)).
		WithButton(skipButton(stepCloudInstalledApp)).
		WithButton(cancelButton)
}

func (p *Plugin) stepCloudUploadApp() flow.Step {
	return flow.NewStep(stepCloudUploadApp).
		WithTitle("Upload the Mattermost app (atlassian-config) to Jira.").
		WithText("To finish the configuration, create a new app in your Jira instance.\n"+
			"Complete the following steps, then come back here and select **Continue**.\n\n"+
			"1. From [**Settings > Apps > Manage Apps**]({{.JiraURL}}/plugins/servlet/upm?source=side_nav_manage_addons) select **Upload app** (see attached screenshot).\n"+
			"2. In the **From this URL field**, enter: `{{.ACURL}}` [link]({{.ACURL}}), then select **Upload**\n"+
			"3. Wait for the app to install. Once completed, you should see an \"Installed and ready to go!\" message.\n").
		WithImage(p.GetPluginURL(), "public/cloud-upload-app.png").
		WithButton(flow.Button{
			Name:     "Waiting for confirmation...",
			Color:    flow.ColorDefault,
			Disabled: true,
		})
}

func (p *Plugin) stepCloudInstalledApp() flow.Step {
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
	return flow.NewStep(stepCloudInstalledApp).
		WithTitle("Confirmed.").
		WithText("You've finished configuring the Mattermost App in Jira. Select **Continue** to setup the subscription webhook.").
		WithButton(flow.Button{
			Name:    "Continue",
			Color:   flow.ColorPrimary,
			OnClick: next(stepWebhook),
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepWebhook() flow.Step {
	return flow.NewStep(stepWebhook).
		WithPretext("##### :white_check_mark: Step 3: setup Jira subscriptions webhook").
		WithText(`To receive Jira event notifications in Mattermost you need to set up a single "firehose" `+
			"webhook, configured for all possible event triggers that you would like to be pushed into "+
			"Mattermost. The plugin's Channel Subscription feature processes the firehose of data and "+
			"then routes the events to channels and users based on your subscriptions.\n\n"+
			"1. Navigate to [Jira System Settings/Webhooks]({{.ManageWebhooksURL}}) (see _screenshot_), select **Create a WebHook** in the top right corner.\n"+
			"2. Give your webhook a symbolic **Name** of your choice.\n"+
			"3. **Status**: Enabled.\n"+
			"4. Leave **URL** blank for the moment. Once you are done configuring the webhook options, come back "+
			"here and select **View URL** to see the confidential URL.\n"+
			"5. **Issue related events**: we recommend leaving the query at **All Issues**. Check **Comment**, "+
			"**Attachment**, and **Issue** events. We recommend checking all of these boxes. These events will be "+
			"further filtered by Mattermost subscriptions. Leave **Entity property**, **Worklog**, and **Issue "+
			"link** events unchecked, they are not yet supported.\n"+
			"6. Leave all other checkboxes blank.\n"+
			"7. Select **View URL** to see the secret **URL** to enter in Jira, and continue.\n").
		WithImage(p.GetPluginURL(), "public/configure-webhook.png").
		WithButton(flow.Button{
			Name:  "View URL",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "Jira Webhook URL",
				IntroductionText: "Please scroll to select the entire URL if necessary. [link]({{.WebhookURL}})\n```\n{{.WebhookURL}}\n```\nOnce you have entered all options and the webhook URL, select **Create Webhook**",
				SubmitLabel:      "Continue",
			},
			OnDialogSubmit: func(f *flow.Flow, _ map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
				delegatedFrom := f.GetState().GetString(keyDelegatedFromUserID)
				if delegatedFrom != "" {
					_ = p.setupFlow.ForUser(delegatedFrom).Go(stepDelegateComplete)
				}
				return stepWebhookDone, nil, nil, nil
			},
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepWebhookDone() flow.Step {
	return flow.NewStep(stepWebhookDone).
		WithTitle("Webhook setup.").
		WithText("<>/<> TODO how to subscribe.").
		Next(stepConnect)
}

func (p *Plugin) stepConnect() flow.Step {
	return flow.NewStep(stepConnect).
		WithPretext("##### :white_check_mark: Step 4: connect your Jira user account").
		WithText("Go **[here]({{.ConnectURL}})** to connect your account.").
		WithButton(flow.Button{
			Name:     "Waiting for confirmation...",
			Color:    flow.ColorDefault,
			Disabled: true,
		})
}

func (p *Plugin) stepConnected() flow.Step {
	return flow.NewStep(stepConnected).
		WithTitle("Connected Jira user account.").
		WithText("You've connected your user account to Jira.").
		Next(stepDone)
}

func (p *Plugin) stepCancel() flow.Step {
	return flow.NewStep(stepCancel).
		Terminal().
		WithPretext("##### :no_entry_sign: Canceled").
		WithText("<>/<> TODO how to finish manually.")
}

func (p *Plugin) stepDone() flow.Step {
	return flow.NewStep(stepDone).
		Terminal().
		WithPretext("##### :wave: All done!").
		WithTitle("The Jira integration is now fully configured.").
		WithText("<>/<> TODO next steps.")
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

func (p *Plugin) submitCreateCloudInstance(f *flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, nil, errors.New("no Jira cloud URL in the request")
	}
	jiraURL = strings.TrimSpace(jiraURL)
	if jiraOrgRegexp.MatchString(jiraURL) {
		jiraURL = fmt.Sprintf("https://%s.atlassian.net", jiraURL)
	}

	jiraURL, err := p.installInactiveCloudInstance(jiraURL, f.UserID)
	if err != nil {
		return "", nil, nil, err
	}

	return stepCloudAddedInstance, flow.State{
		keyEdition:             string(CloudInstanceType),
		keyJiraURL:             jiraURL,
		keyAtlassianConnectURL: p.GetPluginURL() + instancePath(routeACJSON, types.ID(jiraURL)),
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
