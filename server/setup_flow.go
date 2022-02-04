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
	stepDelegate                 flow.Name = "delegate"
	stepDelegateComplete         flow.Name = "delegate-complete"
	stepDelegated                flow.Name = "delegated"
	stepChooseEdition            flow.Name = "choose-edition"
	stepCloudAddedInstance       flow.Name = "cloud-added"
	stepCloudEnableDeveloperMode flow.Name = "cloud-enable-dev"
	stepCloudUploadApp           flow.Name = "cloud-upload-app"
	stepCloudInstalledApp        flow.Name = "cloud-installed"
	stepServerAddAppLink         flow.Name = "server-add-link"
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
			p.stepDelegate(),
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
			p.stepServerConfigureAppLink1(),
			p.stepServerConfigureAppLink2(),

			p.stepConnect(),
			p.stepConnected(),

			p.stepWebhook(),
			p.stepWebhookDone(),

			p.stepCancel(),
			p.stepDone(),
		).
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
			"- **Step 1:** <>/<> TODO: describe the steps....\n").
		WithButton(continueButton(stepDelegate)).
		WithButton(cancelButton)
}

func (p *Plugin) stepDelegate() flow.Step {
	return flow.NewStep(stepDelegate).
		WithPretext("##### :hand: Are you a Jira administrator?").
		WithText("Configuring the integration requires administrator access to Jira. If you aren't a Jira admin you can ask another Mattermost user to do it.").
		WithButton(flow.Button{
			Name:    "Continue myself",
			Color:   flow.ColorPrimary,
			OnClick: flow.Goto(stepChooseEdition),
		}).
		WithButton(flow.Button{
			Name:  "I need someone else",
			Color: flow.ColorDefault,
			Dialog: &model.Dialog{
				Title:       "Send instructions",
				SubmitLabel: "Send",
				Elements: []model.DialogElement{
					{
						DisplayName: "", // TODO: This will still show a *
						Name:        "aider",
						Type:        "select",
						DataSource:  "users",
						Placeholder: "Search for people",
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
		Next(stepDone)
}

func (p *Plugin) stepChooseEdition() flow.Step {
	return flow.NewStep(stepChooseEdition).
		WithPretext("##### :white_check_mark: Choose Jira Edition.").
		WithTitle("Cloud or Server (on-premise).").
		WithText("Please choose whether you use the Atlassian Jira Cloud or Server (on-premise) edition. " +
			"If you need to integrate with more than one Jira instance, please refer to the [documentation](<>/<> TODO)").
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
		WithPretext("##### :white_check_mark: Configure the Mattermost application link in Jira").
		WithTitle("Create Application Link.").
		WithText("Server `{{.JiraURL}}` has been successfully added. To finish the configuration add and configure an Application Link in your Jira instance following these steps:\n" +
			"1. Navigate to [**Settings > Applications > Application Links**]({{.JiraURL}}/plugins/servlet/applinks/listApplicationLinks)\n" +
			"2. Enter `{{.PluginURL}}` as the application link, then select **Create new link**." +
			"3. Ignore any errors in Jira's **Configure Application URL** confirmation screen, and select **Continue**.").
		WithButton(continueButton(stepServerConfigureAppLink1)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfigureAppLink1() flow.Step {
	return flow.NewStep(stepServerConfigureAppLink1).
		WithTitle("Create Incoming Application Link.").
		WithText("1. In Jira's **Link Applications** screen enter the following values:\n" +
			"  - **Application Name**:  `Mattermost`\n" +
			"  - **Application Type**: **Generic Application**\n" +
			"  - _other fields_...: _blank_\n" +
			"  - **Create incoming link**: :heavy_check_mark: **IMPORTANT: check**\n" +
			"2. Select **Continue**\n").
		WithButton(continueButton(stepServerConfigureAppLink2)).
		WithButton(cancelButton)
}

func (p *Plugin) stepServerConfigureAppLink2() flow.Step {
	return flow.NewStep(stepServerConfigureAppLink2).
		WithTitle("Configure Incoming Application Link.").
		WithText("1. In Jira's second **Link Applications** screen enter the following values:\n" +
			"  - **Consumer Key**: `{{.MattermostKey}}`\n" +
			"  - **Consumer Name**: `Mattermost`\n" +
			"  - **Public Key**:\n```\n{{ .PublicKey }}\n```\n" +
			"  - **Consumer Callback URL**: _leave blank_\n" +
			"  - **Allow 2-legged OAuth**: off (unchecked) \n" +
			"2. Select **Continue**\n").
		WithButton(continueButton(stepConnect)).
		WithButton(cancelButton)
}

func (p *Plugin) stepCloudAddedInstance() flow.Step {
	return flow.NewStep(stepCloudAddedInstance).
		WithText("Jira cloud `{{.JiraURL}}` has been added, and is ready to configure.").
		Next(stepCloudEnableDeveloperMode)
}

func (p *Plugin) stepCloudEnableDeveloperMode() flow.Step {
	return flow.NewStep(stepCloudEnableDeveloperMode).
		WithPretext("##### :white_check_mark: Configure the Mattermost app in Jira").
		WithTitle("Enable development mode.").
		WithText("Mattermost Jira Cloud integration requires setting your Jira to _development mode_. " +
			"Enabling the development mode allows you to install apps like Mattermost, from outside the Atlassian Marketplace." +
			"Complete the following steps, then select **Continue**:\n\n" +
			"1. Navigate to [**Settings > Apps > Manage Apps**]({{.JiraURL}}/plugins/servlet/upm?source=side_nav_manage_addons).\n" +
			"2. Select **Settings** at the bottom of the page.\n" +
			"3. Select **Enable development mode**, then select **Apply**.\n").
		WithButton(continueButton(stepCloudUploadApp)).
		WithButton(skipButton(stepCloudInstalledApp)).
		WithButton(cancelButton)
}

func (p *Plugin) stepCloudUploadApp() flow.Step {
	return flow.NewStep(stepCloudUploadApp).
		WithTitle("Upload the Mattermost app (atlassian-config) to Jira.").
		WithText("To finish the configuration, create a new app in your Jira instance by following these steps:\n\n" +
			"1. From [**Settings > Apps > Manage Apps**]({{.JiraURL}}/plugins/servlet/upm?source=side_nav_manage_addons) select **Upload app**.\n" +
			"2. In the **From this URL field**, enter: `{{.ACURL}}`, then select **Upload**\n" +
			"3. Wait for the app to install. Once completed, you should see an \"Installed and ready to go!\" message.\n").
		WithButton(flow.Button{
			Name:     "Waiting for confirmation...",
			Color:    flow.ColorDefault,
			Disabled: true,
		})
}

func (p *Plugin) stepCloudInstalledApp() flow.Step {
	next := func(to flow.Name) func(*flow.Flow) (flow.Name, flow.State, error) {
		return func(f *flow.Flow) (flow.Name, flow.State, error) {
			jiraURL := f.State[keyJiraURL]
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
		WithText("You've finished configuring the Mattermost App in Jira. Select **Continue** to connect your user account.").
		WithButton(flow.Button{
			Name:    "Continue",
			Color:   flow.ColorPrimary,
			OnClick: next(stepConnect),
		}).
		WithButton(flow.Button{
			Name:    "DEBUG Skip to webhook",
			Color:   flow.ColorWarning,
			OnClick: next(stepWebhook),
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepConnect() flow.Step {
	return flow.NewStep(stepConnect).
		WithPretext("##### :white_check_mark: Connect your Jira user account").
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
		Next(stepWebhook)
}

func (p *Plugin) stepWebhook() flow.Step {
	return flow.NewStep(stepWebhook).
		WithPretext("##### :white_check_mark: Setup Jira subscriptions webhook").
		WithText("Navigate to [Jira System Settings/Webhooks]({{.ManageWebhooksURL}}) where you can create it. " +
			"The webhook needs to be set up once, is shared by all channels and subscription filters.\n" +
			"<>/<> TODO more about how to set up sub webhook + screenshots\n" +
			"Click **View URL** to see the secret **URL** to enter in Jira. You can use `/jira webhook` command to see the secret URL again later.\n").
		WithButton(flow.Button{
			Name:  "View URL",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "Jira Webhook URL",
				IntroductionText: "Please scroll to select the entire URL if necessary:\n```\n{{.WebhookURL}}\n```\n",
				SubmitLabel:      "Continue",
			},
			OnDialogSubmit: func(f *flow.Flow, _ map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
				delegatedFrom, ok := f.State[keyDelegatedFromUserID]
				if ok {
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
		keyPublicKey:         strings.TrimSpace(string(pkey)),
		keyConnectURL:        p.GetPluginURL() + "/" + instancePath(routeUserConnect, si.InstanceID),
		keyWebhookURL:        p.getSubscriptionsWebhookURL(si.InstanceID),
		keyManageWebhooksURL: si.GetManageWebhooksURL(),
	}, nil, nil
}
