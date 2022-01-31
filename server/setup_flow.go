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
	stepDelegated                flow.Name = "delegated"
	stepChooseEdition            flow.Name = "choose-edition"
	stepCloudAddedInstance       flow.Name = "cloud-added"
	stepCloudEnableDeveloperMode flow.Name = "cloud-enable-dev"
	stepCloudUploadApp           flow.Name = "cloud-upload-app"
	stepCloudInstalledApp        flow.Name = "cloud-installed"
	stepCloudConnect             flow.Name = "cloud-connect"
	stepCloudConnected           flow.Name = "cloud-connected"
	stepConfigureServerApp       flow.Name = "configure-server-app"
	stepWebhook                  flow.Name = "webhook"
	stepWebhookDone              flow.Name = "webhook-done"
	stepCancel                   flow.Name = "cancel"
	stepDone                     flow.Name = "done"
)

const (
	keyEdition             = "Edition"
	keyDelegatedTo         = "Delegated"
	keyJiraURL             = "URL"
	keyConnectURL          = "ConnectURL"
	keyWebhookURL          = "WebhookURL"
	keyManageWebhooksURL   = "ManageWebhooksURL"
	keyAtlassianConnectURL = "ACURL"
)

func (p *Plugin) NewSetupFlow() flow.Flow {
	pluginURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + manifest.ID
	conf := p.getConfig()
	return flow.NewUserFlow("setup-wizard", p.client, pluginURL, conf.botUserID).
		WithSteps(
			p.stepWelcome(),
			p.stepDelegate(),
			p.stepDelegated(),
			p.stepChooseEdition(),

			// Jira Cloud steps
			p.stepCloudAddedInstance(),
			p.stepCloudEnableDeveloperMode(),
			p.stepCloudUploadApp(),
			p.stepCloudInstalledApp(),

			p.stepCloudConnect(),
			p.stepCloudConnected(),

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

func continueButtonF(f func(f flow.Flow) (flow.Name, flow.State)) flow.Button {
	return flow.Button{
		Name:    "Continue",
		Color:   flow.ColorPrimary,
		OnClick: f,
	}
}

func continueButton(next flow.Name) flow.Button {
	return continueButtonF(flow.Goto(next))
}

func skipButtonF(f func(f flow.Flow) (flow.Name, flow.State)) flow.Button {
	return flow.Button{
		Name:    "DEBUG Skip",
		Color:   flow.ColorWarning,
		OnClick: f,
	}
}

func skipButton(next flow.Name) flow.Button {
	return skipButtonF(flow.Goto(next))
}

func (p *Plugin) stepWelcome() flow.Step {
	return flow.NewStep(stepSetupWelcome).
		WithPretext("##### :wave: Welcome to Jira integration! [Learn more](https://github.com/mattermost/mattermost-plugin-jira#readme)").
		WithTitle("Configure the integration.").
		WithMessage("Just a few more steps to go!\n" +
			"- **Step 1:** <>/<> TODO.\n").
		WithButton(continueButton(stepDelegate)).
		WithButton(cancelButton)
}

func (p *Plugin) stepDelegate() flow.Step {
	return flow.NewStep(stepDelegate).
		WithPretext("##### :hand: Are you a Jira administrator?").
		WithMessage("Configuring the integration requires administrator access to Jira. If you are not a Jira administrator you can ask another Mattermost user to do it.").
		WithButton(flow.Button{
			Name:    "Continue myself",
			Color:   flow.ColorPrimary,
			OnClick: flow.Goto(stepChooseEdition),
		}).
		WithButton(flow.Button{
			Name:  "Ask someone else",
			Color: flow.ColorDefault,
			Dialog: &model.Dialog{
				Title:       "Send instructions to",
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

func (p *Plugin) submitDelegateSelection(f flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, string, map[string]string) {
	aiderIDRaw, ok := submission["aider"]
	if !ok {
		return "", nil, "aider missing", nil
	}
	aiderID, ok := aiderIDRaw.(string)
	if !ok {
		return "", nil, "aider is not a string", nil
	}

	aider, err := p.client.User.Get(aiderID)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed get user").Error(), nil
	}

	// <>/<> TODO add and test the delegate flow
	// err = p.StartSetupWizard(aider.Id, true)
	// if err != nil {
	// 	return 0, nil, errors.Wrap(err, "failed start configration wizzard").Error(), nil
	// }

	return stepDelegated, flow.State{
		keyDelegatedTo: aider.Id,
	}, "", nil
}

func (p *Plugin) stepDelegated() flow.Step {
	return flow.NewStep(stepDelegated).
		WithMessage("Asked {{.Delegated}} to finish configuring the integration").
		WithButton(cancelButton)
}

func (p *Plugin) stepChooseEdition() flow.Step {
	return flow.NewStep(stepChooseEdition).
		WithPretext("##### :white_check_mark: Choose Jira Edition.").
		WithTitle("Cloud or Server (on-premise).").
		WithMessage("Please choose whether you use the Atlassian Jira Cloud or Server (on-premise) edition. " +
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
						SubType:     "text",
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
						SubType:     "text",
					},
				},
			},
			OnDialogSubmit: p.submitCreateServerInstance,
		}).
		WithButton(cancelButton)
}

var jiraOrgRegexp = regexp.MustCompile(`^[\w-]+$`)

func (p *Plugin) submitCreateCloudInstance(f flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, "no Jira cloud URL in the request", nil
	}
	jiraURL = strings.TrimSpace(jiraURL)
	if jiraOrgRegexp.MatchString(jiraURL) {
		jiraURL = fmt.Sprintf("https://%s.atlassian.net", jiraURL)
	}

	_, userID := f.State()

	jiraURL, err := p.installInactiveCloudInstance(jiraURL, userID)
	if err != nil {
		return "", nil, err.Error(), nil
	}

	return stepCloudAddedInstance, flow.State{
		keyEdition:             string(CloudInstanceType),
		keyJiraURL:             jiraURL,
		keyAtlassianConnectURL: p.GetPluginURL() + instancePath(routeACJSON, types.ID(jiraURL)),
	}, "", nil
}

func (p *Plugin) submitCreateServerInstance(f flow.Flow, submission map[string]interface{}) (flow.Name, flow.State, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, "no Jira server URL in the request", nil
	}
	jiraURL = strings.TrimSpace(jiraURL)

	_, _, err := p.installServerInstance(jiraURL)
	if err != nil {
		return "", nil, err.Error(), nil
	}
	// <>/<> TODO add Jira Server flow
	return stepConfigureServerApp, flow.State{
		keyEdition: string(ServerInstanceType),
		keyJiraURL: jiraURL,
	}, "", nil
}

func (p *Plugin) stepCloudAddedInstance() flow.Step {
	return flow.NewStep(stepCloudAddedInstance).
		WithMessage("Jira cloud {{.URL}} has been added, and is ready to configure.")
}

func (p *Plugin) stepCloudEnableDeveloperMode() flow.Step {
	return flow.NewStep(stepCloudEnableDeveloperMode).
		WithPretext("##### :white_check_mark: Configure the Mattermost app in Jira").
		WithTitle("Enable development mode.").
		WithMessage("Mattermost Jira Cloud integration requires setting your Jira to _development mode_. " +
			"Enabling the development mode allows you to install apps like Mattermost, from outside the Atlassian Marketplace." +
			"Please follow these steps and press **Continue** when done:\n\n" +
			"1. Navigate to [**Settings > Apps > Manage Apps**]({{.URL}}/plugins/servlet/upm?source=side_nav_manage_addons).\n" +
			"2. Click **Settings** at bottom of page.\n" +
			"3. Check **Enable development mode**, and press **Apply**.\n").
		WithButton(continueButton(stepCloudUploadApp)).
		WithButton(skipButton(stepCloudInstalledApp)).
		WithButton(cancelButton)
}

func (p *Plugin) stepCloudUploadApp() flow.Step {
	return flow.NewStep(stepCloudUploadApp).
		WithTitle("Upload Mattermost app (atlassian-config) to Jira.").
		WithMessage("To finish the configuration, create a new app in your Jira instance by following these steps:\n\n" +
			"1. From [**Settings > Apps > Manage Apps**]({{.URL}}/plugins/servlet/upm?source=side_nav_manage_addons) click **Upload app**.\n" +
			"2. In the **From this URL field**, enter: `{{.ACURL}}`, press **Upload**\n" +
			"3. Wait for the app to install. Once completed, you should see an \"Installed and ready to go!\" message.\n").
		WithButton(flow.Button{
			Name:     "Waiting for confirmation...",
			Color:    flow.ColorDefault,
			Disabled: true,
		})
}

func (p *Plugin) stepCloudInstalledApp() flow.Step {
	next := func(to flow.Name) func(flow.Flow) (flow.Name, flow.State) {
		return func(f flow.Flow) (flow.Name, flow.State) {
			state, _ := f.State()
			jiraURL := state[keyJiraURL]
			instanceID := types.ID(jiraURL)
			return to, flow.State{
				keyConnectURL:        p.GetPluginURL() + "/" + instancePath(routeUserConnect, instanceID),
				keyWebhookURL:        p.getSubscriptionsWebhookURL(instanceID),
				keyManageWebhooksURL: cloudManageWebhooksURL(jiraURL),
			}
		}
	}
	return flow.NewStep(stepCloudInstalledApp).
		WithTitle("Confirmed.").
		WithMessage("You have finished configuring the Mattermost App in Jira. Please select **Continue** to connect your user account.").
		WithButton(flow.Button{
			Name:    "Continue",
			Color:   flow.ColorPrimary,
			OnClick: next(stepCloudConnect),
		}).
		WithButton(flow.Button{
			Name:    "DEBUG Skip to webhook",
			Color:   flow.ColorWarning,
			OnClick: next(stepWebhook),
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepCloudConnect() flow.Step {
	return flow.NewStep(stepCloudConnect).
		WithPretext("##### :white_check_mark: Connect your Jira user account").
		WithMessage("Go **[here]({{.ConnectURL}})** to connect your account.").
		WithButton(flow.Button{
			Name:     "Waiting for confirmation...",
			Color:    flow.ColorDefault,
			Disabled: true,
		})
}

func (p *Plugin) stepCloudConnected() flow.Step {
	return flow.NewStep(stepCloudConnected).
		WithTitle("Connected Jira user account.").
		WithMessage("You have connected your user account to Jira.")
}

func (p *Plugin) stepWebhook() flow.Step {
	return flow.NewStep(stepWebhook).
		WithPretext("##### :white_check_mark: Setup Jira subscriptions webhook").
		WithMessage("Navigate to [Jira System Settings/Webhooks]({{.ManageWebhooksURL}}) where you can create it. " +
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
			OnDialogSubmit: flow.DialogGoto(stepWebhookDone),
		}).
		WithButton(cancelButton)
}

func (p *Plugin) stepWebhookDone() flow.Step {
	return flow.NewStep(stepWebhookDone).
		WithTitle("Webhook setup.").
		WithMessage("<>/<> TODO how to subscribe.").
		WithButton(flow.Button{
			Name:    "Done",
			Color:   flow.ColorPrimary,
			OnClick: flow.Goto(stepDone),
		})
}

func (p *Plugin) stepCancel() flow.Step {
	return flow.NewStep(stepCancel).
		Terminal().
		WithPretext("##### :no_entry_sign: Canceled").
		WithMessage("<>/<> TODO how to finish manually.")
}

func (p *Plugin) stepDone() flow.Step {
	return flow.NewStep(stepDone).
		Terminal().
		WithPretext("##### :wave: All done!").
		WithTitle("The Jira integration is now fully configured.").
		WithMessage("<>/<> TODO next steps.")
}
