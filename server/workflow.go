// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-workflow-client/workflowclient"
)

type WorkflowTrigger struct {
	SubscriptionFilters
}
type TriggerStore map[string]WorkflowTrigger

func (t TriggerStore) AddTrigger(trigger WorkflowTrigger, callbackURL string) {
	t[callbackURL] = trigger
}

func (t TriggerStore) RemoveTrigger(callbackURL string) {
	delete(t, callbackURL)
}

func (p *Plugin) NotifyWorkflow(wh *webhook) error {
	activateParams := workflowclient.ActivateParameters{
		TriggerVars: map[string]string{
			"Summary":     wh.headline,
			"Description": wh.text,
		},
	}

	callbacks := []string{}
	for callbackURL, trigger := range p.workflowTriggerStore {
		if !p.matchesSubsciptionFilters(wh, trigger.SubscriptionFilters) {
			continue
		}

		callbacks = append(callbacks, callbackURL)
	}

	if err := workflowclient.NewClientPlugin(p.API).WorkflowCallbacks(callbacks, activateParams); err != nil {
		return fmt.Errorf("Unable to notify some workflows: %w", err)
	}

	return nil
}
