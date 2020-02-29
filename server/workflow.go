// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"sync"

	"github.com/mattermost/mattermost-plugin-workflow-client/workflowclient"
)

type WorkflowTrigger struct {
	SubscriptionFilters
}
type TriggerStore struct {
	store map[string]WorkflowTrigger
	lock  sync.RWMutex
}

func NewTriggerStore() *TriggerStore {
	return &TriggerStore{
		store: make(map[string]WorkflowTrigger),
	}
}

func (t *TriggerStore) AddTrigger(trigger WorkflowTrigger, callbackURL string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.store[callbackURL] = trigger
}

func (t *TriggerStore) RemoveTrigger(callbackURL string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	delete(t.store, callbackURL)
}

func (t *TriggerStore) ForEach(forEach func(string, WorkflowTrigger)) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	for callbackURL, trigger := range t.store {
		forEach(callbackURL, trigger)
	}
}

func (p *Plugin) NotifyWorkflow(wh *webhook) error {
	activateParams := workflowclient.ActivateParameters{
		TriggerVars: map[string]string{
			"Summary":     wh.Issue.Fields.Summary,
			"Description": wh.text,
			"Headline":    wh.headline,
			"Key":         wh.Issue.Key,
			"ID":          wh.Issue.ID,
		},
	}

	callbacks := []string{}
	p.workflowTriggerStore.ForEach(func(callbackURL string, trigger WorkflowTrigger) {
		if !p.matchesSubsciptionFilters(wh, trigger.SubscriptionFilters) {
			return
		}
		callbacks = append(callbacks, callbackURL)
	})

	if err := workflowclient.NewClientPlugin(p.API).NotifyWorkflows(callbacks, activateParams); err != nil {
		return fmt.Errorf("Unable to notify some workflows: %w", err)
	}

	return nil
}
