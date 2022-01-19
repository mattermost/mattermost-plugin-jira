package main

import (
	"net/url"
	"strings"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow/steps"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	FlowStepWelcome                = "welcome"
	FlowStepChooseEdition          = "choose_edition"
	FlowStepConfirmNonAtlassianURL = "confirm-non-atlassian"
	FlowStepTestNext1              = "test-next1"
	FlowStepTestNext2              = "test-next2"
	FlowStepCancel                 = "cancel"
)

type FlowManager struct {
	client           *pluginapi.Client
	getConfiguration func() config
	pluginURL        string

	logger logger.Logger
	poster poster.Poster
	store  flow.Store
	ctl    flow.Controller
	index  map[string]int
	steps  []steps.Step
}

func (p *Plugin) NewFlowManager() *FlowManager {
	conf := p.getConfig()

	fm := &FlowManager{
		client:           p.client,
		logger:           p.log,
		pluginURL:        *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + manifest.ID,
		poster:           poster.NewPoster(&p.client.Post, conf.botUserID),
		store:            flow.NewFlowStore(*p.client, "flow_store"),
		getConfiguration: p.getConfig,
		index:            map[string]int{},
	}

	fm.addStep(FlowStepWelcome, fm.stepWelcome())
	fm.addStep(FlowStepChooseEdition, fm.stepChooseEdition())
	fm.addStep(FlowStepConfirmNonAtlassianURL, steps.NewEmptyStep("", "<>/<> TODO Confirm Non-Atlassian URL"))
	fm.addStep(FlowStepTestNext1, steps.NewEmptyStep("", "<>/<> TODO Next 1"))
	fm.addStep(FlowStepTestNext2, steps.NewEmptyStep("", "<>/<> TODO Next 2"))
	fm.addStep(FlowStepCancel, steps.NewEmptyStep("", "<>/<> TODO Finished"))

	fm.ctl = flow.NewFlowController(
		fm.logger,
		p.gorillaRouter,
		fm.poster,
		&p.client.Frontend,
		fm.pluginURL,
		flow.NewFlow(fm.steps, routeSetupWizard, nil),
		fm.store,
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
			OnClick: func() int {
				return fm.skipOffset(FlowStepWelcome, FlowStepChooseEdition)
			},
		}).
		WithButton(steps.Button{
			Name:  "Cancel",
			Style: steps.Danger,
			OnClick: func() int {
				return fm.skipOffset(FlowStepWelcome, FlowStepCancel)
			},
		}).
		Build()
}

func (fm *FlowManager) stepChooseEdition() steps.Step {
	pretext := fm.pretext(":white_check_mark: Choose Jira Edition - Cloud or Server")
	text := "Please choose whether you use the Atlassian Jira Cloud or Server (on-prem) edition. "
	text += "If you need to integrate with more than one Jira instance, please refer to the [documentation](<>/<> TODO)"
	return steps.NewCustomStepBuilder("", text).
		WithPretext(pretext).
		WithButton(steps.Button{
			Name:  "Jira Cloud",
			Style: steps.Primary,
			Dialog: &steps.Dialog{
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
			},
			OnClick: func() int {
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
			OnClick: func() int {
				return -1
			},
		}).
		WithButton(steps.Button{
			Name:  "Cancel",
			Style: steps.Danger,
			OnClick: func() int {
				return fm.skipOffset(FlowStepChooseEdition, FlowStepCancel)
			},
		}).
		Build()
}

func (fm *FlowManager) submitCreateCloudInstance(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return 0, nil, "no URL in the request", nil
	}
	u, err := url.Parse(jiraURL)
	switch {
	case err == nil && !strings.HasSuffix(u.Host, "atlassian.net"):
		return fm.skipOffset(FlowStepChooseEdition, FlowStepConfirmNonAtlassianURL), nil, "", nil

	case err == nil:
		//

	default:
		return 0, nil, err.Error(), nil
	}

	err = fm.submitCreateInstance(false, userID, u.String())
	if err != nil {
		return 0, nil, err.Error(), nil
	}

	return fm.skipOffset(FlowStepChooseEdition, FlowStepTestNext1), nil, "", nil
}

func (fm *FlowManager) submitCreateServerInstance(userID string, submission map[string]interface{}) (int, *steps.Attachment, string, map[string]string) {
	jiraURL, _ := submission["url"].(string)
	if jiraURL == "" {
		return 0, nil, "no URL in the request", nil
	}

	err := fm.submitCreateInstance(false, userID, jiraURL)
	if err != nil {
		return 0, nil, err.Error(), nil
	}

	return fm.skipOffset(FlowStepChooseEdition, FlowStepTestNext2), nil, "", nil
}

func (fm *FlowManager) submitCreateInstance(isServer bool, userID, jiraURL string) error {

	// TODO Validate URL? save Config

	fm.logger.Errorf("<>/<> %s\n", jiraURL)

	return nil
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

func (fm *FlowManager) skipOffset(fromName, toName string) int {
	from, ok := fm.index[fromName]
	if !ok {
		return -1
	}
	to, ok := fm.index[toName]
	if !ok {
		return -1
	}
	return to - from - 1
}
