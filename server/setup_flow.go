package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow/steps"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	FlowStepWelcome                = "welcome"
	FlowStepChooseEdition          = "choose_edition"
	FlowStepConfirmNonAtlassianURL = "confirm-non-atlassian"
	FlowStepConfigureCloudApp      = "configure-cloud-app"
	FlowStepConfigureServerApp     = "configure-server-app"
	FlowStepTestNext1              = "test-next1"
	FlowStepTestNext2              = "test-next2"
	FlowStepCancel                 = "cancel"
)

type FlowManager struct {
	p     *Plugin
	ctl   flow.Controller
	index map[string]int
	steps []steps.Step
}

func (p *Plugin) NewFlowManager() *FlowManager {
	fm := &FlowManager{
		p:     p,
		index: map[string]int{},
	}
	fm.addStep(FlowStepWelcome, fm.stepWelcome())
	fm.addStep(FlowStepChooseEdition, fm.stepChooseEdition())
	fm.addStep(FlowStepConfirmNonAtlassianURL, fm.stepConfirmNonAtlassianURL(""))
	fm.addStep(FlowStepCloudURLError, fm.stepConfirmNonAtlassianURL(""))
	fm.addStep(FlowStepConfigureCloudApp, fm.stepConfigureCloudApp(""))
	fm.addStep(FlowStepTestNext1, steps.NewEmptyStep("", "<>/<> TODO Next 1"))
	fm.addStep(FlowStepTestNext2, steps.NewEmptyStep("", "<>/<> TODO Next 2"))
	fm.addStep(FlowStepCancel, steps.NewEmptyStep("", "<>/<> TODO Finished"))

	conf := p.getConfig()
	pluginURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + manifest.ID
	fm.ctl = flow.NewFlowController(
		p.log,
		p.gorillaRouter,
		poster.NewPoster(&p.client.Post, conf.botUserID),
		&p.client.Frontend,
		pluginURL,
		flow.NewFlow(fm.steps, routeSetupWizard, nil),
		flow.NewFlowStore(*p.client, "flow_store"),
		&propertyStore{},
	)

	return fm
}

func (fm *FlowManager) StartConfigurationWizard(userID string) error {
	err := fm.ctl.Start(userID)
	if err != nil {
		return err
	}

	return nil
}

func (fm *FlowManager) stepWelcome() steps.Step {
	pretext := fm.pretext(":wave: Welcome to Jira for Mattermost!")
	text := "Configure the Mattermost Jira integration. This step requires Mattermost site administrator access. It also requires involvement from your organization's Jira administrator. If you do not have administrator access to Jira, please find out who can help you with the setup before proceeding further."
	return steps.NewCustomStepBuilder("", text).
		WithPretext(pretext).
		WithButton(steps.Button{
			Name:  "Continue",
			Style: steps.Primary,
			OnClick: func(userID string) int {
				return fm.skip(userID, FlowStepChooseEdition)
			},
		}).
		WithButton(steps.Button{
			Name:  "Cancel",
			Style: steps.Danger,
			OnClick: func(userID string) int {
				return fm.skip(userID, FlowStepCancel)
			},
		}).
		Build()
}

func (fm *FlowManager) dialogEnterJiraCloudURL() *steps.Dialog {
	return &steps.Dialog{
		Dialog: model.Dialog{
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
		OnDialogSubmit: fm.submitCreateCloudInstance,
	}
}

func (fm *FlowManager) stepChooseEdition() steps.Step {
	pretext := fm.pretext(":white_check_mark: Choose Jira Edition - Cloud or Server")
	text := "Please choose whether you use the Atlassian Jira Cloud or Server (on-prem) edition. "
	text += "If you need to integrate with more than one Jira instance, please refer to the [documentation](<>/<> TODO)"
	return steps.NewCustomStepBuilder("", text).
		WithPretext(pretext).
		WithButton(steps.Button{
			Name:   "Jira Cloud",
			Style:  steps.Primary,
			Dialog: fm.dialogEnterJiraCloudURL(),
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Jira Server",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
				Dialog: model.Dialog{
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
				OnDialogSubmit: fm.submitCreateServerInstance,
			},
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Cancel",
			Style: steps.Danger,
			OnClick: func(userID string) int {
				return fm.skip(userID, FlowStepCancel)
			},
		}).
		Build()
}

func (fm *FlowManager) stepConfirmNonAtlassianURL(jiraURL string) steps.Step {
	pretext := fm.pretext(":warning: Confirm your Jira Cloud URL")
	text := fmt.Sprintf("The URL you entered (`%s`) does not look like a Jira Cloud URL, they usually look like `yourorg.atlassian.net`.", jiraURL)
	return steps.NewCustomStepBuilder("", text).
		WithPretext(pretext).
		WithButton(steps.Button{
			Name:  "Confirm and Continue",
			Style: steps.Primary,
			OnClick: func(userID string) int {
				// TODO: <>/<> need persistent state, can't really override
				// OnClick, this only works on a single server, and because ctl
				// stores a pointer to steps.
				jiraURL, err := fm.p.installInactiveCloudInstance(jiraURL)
				if err != nil {
					panic(err.Error())
					return -1
				}
				return fm.overrideAndSkip(userID, FlowStepConfigureCloudApp, fm.stepConfigureCloudApp(jiraURL))
			},
		}).
		WithButton(steps.Button{
			Name:   "Re-enter",
			Style:  steps.Primary,
			Dialog: fm.dialogEnterJiraCloudURL(),
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Cancel",
			Style: steps.Danger,
			OnClick: func(userID string) int {
				return fm.skip(userID, FlowStepCancel)
			},
		}).
		Build()
}

func (fm *FlowManager) stepConfigureCloudApp(jiraURL string) steps.Step {
	pretext := fm.pretext(":white_check_mark: Configure Mattermost App in Jira")
	text := fmt.Sprintf("%s has been successfully added. ", jiraURL)
	text += "To finish the configuration, create a new app in your Jira instance following these steps:\n\n"
	text += fmt.Sprintf("1. Navigate to [**Settings > Apps > Manage Apps**](%s/plugins/servlet/upm?source=side_nav_manage_addons).\n", jiraURL)
	text += "2. Click **Settings** at bottom of page, enable development mode, and apply this change.\n"
	text += "  - Enabling development mode allows you to install apps that are not from the Atlassian Marketplace.\n"
	text += "3. Click **Upload app**.\n"
	text += fmt.Sprintf("4. In the **From this URL field**, enter: `%s`", fm.p.GetPluginURL()+instancePath(routeACJSON, types.ID(jiraURL)))
	text += `5. Wait for the app to install. Once completed, you should see an "Installed and ready to go!" message.`

	return steps.NewCustomStepBuilder("", text).
		WithPretext(pretext).
		WithButton(steps.Button{
			Name:  "Continue to Connect Your User Account",
			Style: steps.Primary,
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Ask a Jira Admin",
			Style: steps.Warning,
			OnClick: func(userID string) int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Cancel",
			Style: steps.Danger,
			OnClick: func(userID string) int {
				return fm.skip(userID, FlowStepCancel)
			},
		}).
		Build()
}

var jiraOrgRegexp = regexp.MustCompile(`^[\w-]+$`)

func (fm *FlowManager) submitCreateCloudInstance(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return 0, nil, "no URL in the request", nil
	}
	jiraURL = strings.TrimSpace(jiraURL)
	if jiraOrgRegexp.MatchString(jiraURL) {
		jiraURL = fmt.Sprintf("https://%s.atlassian.net", jiraURL)
	}

	if !utils.IsJiraCloudURL(jiraURL) {
		return fm.overrideAndSkip(userID, FlowStepConfirmNonAtlassianURL, fm.stepConfirmNonAtlassianURL(jiraURL)), nil, "", nil
	}

	jiraURL, err := fm.p.installInactiveCloudInstance(jiraURL)
	if err != nil {
		return 0, nil, err.Error(), nil
	}

	return fm.overrideAndSkip(userID, FlowStepConfigureCloudApp, fm.stepConfigureCloudApp(jiraURL)), nil, "", nil
}

func (fm *FlowManager) submitCreateServerInstance(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return 0, nil, "no URL in the request", nil
	}
	jiraURL = strings.TrimSpace(jiraURL)

	_, _, err := fm.p.installServerInstance(jiraURL)
	if err != nil {
		return 0, nil, err.Error(), nil
	}
	return fm.skip(userID, FlowStepTestNext2), nil, "", nil
}

func (fm FlowManager) pretext(t string) string {
	return "##### " + t
}

type propertyStore struct {
}

func (ps *propertyStore) SetProperty(userID, propertyName string, value interface{}) error {
	return nil
}

func (fm *FlowManager) addStep(name string, step steps.Step) {
	fm.index[name] = len(fm.steps)
	fm.steps = append(fm.steps, step)
}

func (fm *FlowManager) overrideAndSkip(userID, name string, step steps.Step) int {
	i, ok := fm.index[name]
	if !ok {
		return -1
	}
	fm.steps[i] = step

	return fm.skip(userID, name)
}

func (fm *FlowManager) skip(userID, toName string) int {
	_, current, err := fm.ctl.GetCurrentStep(userID)
	if err != nil {
		return -1
	}
	to, ok := fm.index[toName]
	if !ok {
		return -1
	}

	return to - current
}
