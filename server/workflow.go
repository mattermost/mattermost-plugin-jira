// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	triggerapi "github.com/mattermost/mattermost-plugin-workflow/server/trigger/api"
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
	activateParams := triggerapi.ActivateParameters{
		TriggerVars: map[string]string{
			"Summary":     wh.headline,
			"Description": wh.text,
		},
	}

	out, err := json.Marshal(&activateParams)
	if err != nil {
		return err
	}

	for callbackURL, trigger := range p.workflowTriggerStore {
		if !p.matchesSubsciptionFilters(wh, trigger.SubscriptionFilters) {
			continue
		}

		req, err := http.NewRequest("POST", callbackURL, bytes.NewBuffer(out))
		if err != nil {
			return err
		}

		resp := p.API.PluginHTTP(req)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			respBody, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf("Error response from workflow plugin notifying: %v", string(respBody))
		}
	}

	return nil
}
