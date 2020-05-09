// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/pkg/errors"
)

const (
	// Key to migrate the V2 installed instance
	v2keyCurrentJIRAInstance = "current_jira_instance"
	keyInstances             = "known_jira_instances"
	prefixInstance           = "jira_instance_"
)

type Instance interface {
	GetClient(*Connection) (Client, error)
	GetDisplayDetails() map[string]string
	GetUserConnectURL(mattermostUserId string) (string, error)
	GetManageAppsURL() string
	GetURL() string

	Common() *InstanceCommon
	types.Value
}

type InstanceCommon struct {
	*Plugin       `json:"-"`
	PluginVersion string `json:",omitempty"`

	URL       types.ID
	Alias     string
	Type      string
	IsDefault bool
}

func newInstanceCommon(p *Plugin, typ string, url types.ID) *InstanceCommon {
	return &InstanceCommon{
		Plugin:        p,
		Type:          typ,
		URL:           url,
		PluginVersion: manifest.Version,
	}
}

func (common InstanceCommon) GetID() types.ID {
	return common.URL
}

func (common *InstanceCommon) Common() *InstanceCommon {
	return common
}

func (p *Plugin) CreateInactiveCloudInstance(jiraURL string) (returnErr error) {
	ci := newCloudInstance(p, types.ID(jiraURL), false,
		fmt.Sprintf(`{"BaseURL": "%s"}`, jiraURL),
		&AtlassianSecurityContext{BaseURL: jiraURL})
	data, err := json.Marshal(ci)
	if err != nil {
		return errors.WithMessagef(err, "failed to store new Jira Cloud instance:%s", jiraURL)
	}

	// Expire in 15 minutes
	appErr := p.API.KVSetWithExpiry(hashkey(prefixInstance,
		ci.GetURL()), data, 15*60)
	if appErr != nil {
		return errors.WithMessagef(appErr, "failed to store new Jira Cloud instance:%s", jiraURL)
	}
	p.debugf("Stored: new Jira Cloud instance: %s", ci.GetURL())
	return nil
}

func (p *Plugin) LoadInstance(id types.ID) (Instance, error) {
	return p.loadInstance(prefixInstance + id.String())
}

func (p *Plugin) loadInstance(fullkey string) (Instance, error) {
	data, appErr := p.API.KVGet(fullkey)
	if appErr != nil {
		return nil, appErr
	}
	if data == nil {
		return nil, errors.New("not found: " + fullkey)
	}

	// Unmarshal into any of the types just so that we can get the common data
	si := serverInstance{}
	err := json.Unmarshal(data, &si)
	if err != nil {
		return nil, err
	}

	switch si.Type {
	case CloudInstanceType:
		ci := cloudInstance{}
		err = json.Unmarshal(data, &ci)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
		}
		if len(ci.RawAtlassianSecurityContext) > 0 {
			err = json.Unmarshal([]byte(ci.RawAtlassianSecurityContext), &ci.AtlassianSecurityContext)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
			}
		}
		ci.Plugin = p
		return &ci, nil

	case ServerInstanceType:
		si.Plugin = p
		return &si, nil
	}

	return nil, errors.New(fmt.Sprintf("Jira instance %s has unsupported type: %s", fullkey, si.Type))
}

func (p *Plugin) StoreInstance(instance Instance) error {
	store := kvstore.NewStore(kvstore.NewPluginStore(p.API))
	return store.Entity(prefixInstance).Store(instance.GetID(), instance)
}

func (p *Plugin) DeleteInstance(id types.ID) error {
	store := kvstore.NewStore(kvstore.NewPluginStore(p.API))
	return store.Entity(prefixInstance).Delete(id)
}
