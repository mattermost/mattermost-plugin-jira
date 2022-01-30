package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

const (
	stepSetupWelcome            flow.Name = "setup-welcome"
	stepDelegate                          = "delegate"
	stepDelegated                         = "delegated"
	stepChooseEdition                     = "choose-edition"
	stepAddedCloudInstance                = "added-cloud"
	stepEnableJiraDeveloperMode           = "enable-devmode"
	stepUploadJiraApp                     = "upload-ac-app"
	stepConfigureServerApp                = "configure-server-app"
	stepCancel                            = "cancel"
)

const (
	keyEdition             = "Edition"
	keyDelegatedTo         = "Delegated"
	keyJiraURL             = "URL"
	keyAtlassianConnectURL = "ACURL"
)

func (p *Plugin) NewSetupFlow() flow.Flow {
	pluginURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + manifest.ID
	conf := p.getConfig()
	return flow.NewUserFlow("setup", p.client, pluginURL, conf.botUserID).
		WithSteps(
			p.stepWelcome(),
			p.stepDelegate(),
			p.stepDelegated(),
			p.stepChooseEdition(),

			// Jira Cloud steps
			p.stepAddedCloudInstance(),
			p.stepEnableJiraDeveloperMode(),
			p.stepUploadJiraApp(),
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

func (p *Plugin) submitDelegateSelection(userID string, submission map[string]interface{}, state flow.State) (flow.Name, flow.State, string, map[string]string) {
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

	// err = p.StartSetupWizard(aider.Id, true)
	// if err != nil {
	// 	return 0, nil, errors.Wrap(err, "failed start configration wizzard").Error(), nil
	// }

	state[keyDelegatedTo] = aider.Id
	return stepDelegated, state, "", nil
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

func (p *Plugin) submitCreateCloudInstance(userID string, submission map[string]interface{}, state flow.State) (flow.Name, flow.State, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, "no URL in the request", nil
	}
	jiraURL = strings.TrimSpace(jiraURL)
	if jiraOrgRegexp.MatchString(jiraURL) {
		jiraURL = fmt.Sprintf("https://%s.atlassian.net", jiraURL)
	}

	jiraURL, err := p.installInactiveCloudInstance(jiraURL)
	if err != nil {
		return "", nil, err.Error(), nil
	}

	state[keyEdition] = string(CloudInstanceType)
	state[keyJiraURL] = jiraURL
	state[keyAtlassianConnectURL] = p.GetPluginURL() + instancePath(routeACJSON, types.ID(jiraURL))

	return stepEnableJiraDeveloperMode, state, "", nil
}

func (p *Plugin) submitCreateServerInstance(userID string, submission map[string]interface{}, state flow.State) (flow.Name, flow.State, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return "", nil, "no URL in the request", nil
	}
	jiraURL = strings.TrimSpace(jiraURL)

	_, _, err := p.installServerInstance(jiraURL)
	if err != nil {
		return "", nil, err.Error(), nil
	}
	return stepConfigureServerApp, state, "", nil
}

func (p *Plugin) stepAddedCloudInstance() flow.Step {
	return flow.NewStep(stepAddedCloudInstance).
		WithMessage("Jira cloud {{.URL}} has been added, and is ready to configure.")
}

func (p *Plugin) stepEnableJiraDeveloperMode() flow.Step {
	return flow.NewStep(stepEnableJiraDeveloperMode).
		WithPretext("##### :white_check_mark: Configure the Mattermost app in Jira").
		WithTitle("Enable development mode.").
		WithMessage("Mattermost Jira Cloud integration requires setting your Jira to _development mode_. " +
			"Enabling the development mode allows you to install apps like Mattermost, from outside the Atlassian Marketplace." +
			"Please follow these steps and press **Continue** when done:\n\n" +
			"1. Navigate to [**Settings > Apps > Manage Apps**]({{.URL}}/plugins/servlet/upm?source=side_nav_manage_addons).\n" +
			"2. Click **Settings** at bottom of page.\n" +
			"3. Check **Enable development mode**, and press **Apply**.\n").
		WithButton(continueButton(stepUploadJiraApp)).
		WithButton(cancelButton)
}

func (p *Plugin) stepUploadJiraApp() flow.Step {
	return flow.NewStep(stepUploadJiraApp).
		WithTitle("Upload Mattermost app (atlassian-config) to Jira.").
		WithMessage("To finish the configuration, create a new app in your Jira instance by following these steps:\n\n" +
			"1. From [**Settings > Apps > Manage Apps**]({{.URL}}/plugins/servlet/upm?source=side_nav_manage_addons) click **Upload app**.\n" +
			"2. In the **From this URL field**, enter: `{{.ACURL}}`, press **Upload**\n" +
			"3. Wait for the app to install. Once completed, you should see an \"Installed and ready to go!\" message.\n").
		WithButton(continueButton("<>/<> TODO")).
		WithButton(cancelButton)
}
