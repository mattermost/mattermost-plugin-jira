// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

const (
	CloudInstanceType  = "cloud"
	ServerInstanceType = "server"
)

const wSEventInstanceStatus = "instance_status"

type Instances struct {
	*types.ValueSet // of *InstanceCommon, not Instance
	defaultID       types.ID
}

func (instances *Instances) IsEmpty() bool {
	return instances == nil || instances.ValueSet.IsEmpty()
}

func (instances Instances) Get(id types.ID) *InstanceCommon {
	return instances.ValueSet.Get(id).(*InstanceCommon)
}

func (instances Instances) GetDefault() *InstanceCommon {
	if instances.IsEmpty() {
		return nil
	}
	if instances.Len() == 1 {
		return instances.ValueSet.GetAt(0).(*InstanceCommon)
	}
	if instances.defaultID != "" {
		return instances.Get(instances.defaultID)
	}

	for _, id := range instances.ValueSet.IDs() {
		instance := instances.Get(id)
		if instance.IsDefault {
			instances.defaultID = id
			return instance
		}
	}
	instances.defaultID = "not:::a:::url"
	return nil
}

func (instances Instances) SetDefault(id types.ID) error {
	if !instances.Contains(id) {
		return ErrInstanceNotFound
	}

	if instances.defaultID != "" {
		prev := instances.Get(instances.defaultID)
		prev.IsDefault = false
		instances.defaultID = ""
	}

	instance := instances.Get(id)
	instance.IsDefault = true
	instances.defaultID = id
	return nil
}

type instancesCommonArray []*InstanceCommon

func (p instancesCommonArray) Len() int                   { return len(p) }
func (p instancesCommonArray) GetAt(n int) types.Value    { return p[n] }
func (p instancesCommonArray) SetAt(n int, v types.Value) { p[n] = v.(*InstanceCommon) }

func (p instancesCommonArray) InstanceOf() types.ValueArray {
	inst := make(instancesCommonArray, 0)
	return &inst
}
func (p *instancesCommonArray) Ref() interface{} { return &p }
func (p *instancesCommonArray) Resize(n int) {
	*p = make(instancesCommonArray, n)
}

func (p *Plugin) LoadInstances() (*Instances, error) {
	store := kvstore.NewStore(kvstore.NewPluginStore(p.API))
	vs, err := store.ValueIndex(keyInstances, &instancesCommonArray{}).Load()
	if err != nil {
		return nil, err
	}
	return &Instances{
		ValueSet: vs,
	}, nil
}

func (p *Plugin) StoreInstances(instances *Instances) error {
	store := kvstore.NewStore(kvstore.NewPluginStore(p.API))
	return store.ValueIndex(keyInstances, &instancesCommonArray{}).Store(instances.ValueSet)
}

func (p *Plugin) UpdateInstances(updatef func(instances *Instances) error) error {
	instances, err := p.LoadInstances()
	if err != nil {
		return err
	}
	err = updatef(instances)
	if err != nil {
		return err
	}
	return p.StoreInstances(instances)
}

var ErrAlreadyExists = errors.New("already exists")

func (p *Plugin) InstallInstance(instance Instance) error {
	err := p.UpdateInstances(
		func(instances *Instances) error {
			if instances.Contains(instance.Common().URL) {
				return ErrAlreadyExists
			}
			err := p.StoreInstance(instance)
			if err != nil {
				return err
			}
			instances.Set(instance)
			return nil
		})
	if err != nil {
		return err
	}

	// Notify users we have installed an instance
	p.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": true,
			"instance_type":      instance.Common().Type,
		},
		&model.WebsocketBroadcast{},
	)
	return nil
}

var ErrInstanceNotFound = errors.New("instance not found")

func (p *Plugin) UninstallInstance(id types.ID, instanceType string) (Instance, error) {
	var instance Instance
	err := p.UpdateInstances(
		func(instances *Instances) error {
			if !instances.Contains(id) {
				return ErrInstanceNotFound
			}
			var err error
			instance, err = p.LoadInstance(id)
			if err != nil {
				return err
			}
			if instanceType != instance.Common().Type {
				return errors.Errorf("%s did not match instance %s type %s", instanceType, id, instance.Common().Type)
			}
			return p.DeleteInstance(id)
		})
	if err != nil {
		return nil, err
	}

	// Notify users we have uninstalled an instance
	p.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": false,
			"instance_type":      "",
		},
		&model.WebsocketBroadcast{},
	)
	return instance, nil
}

func (p *Plugin) StoreDefaultInstance(id types.ID) error {
	err := p.UpdateInstances(
		func(instances *Instances) error {
			return instances.SetDefault(id)
		})
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) LoadDefaultInstance(explicit types.ID) (Instance, error) {
	id := types.ID("")
	if explicit == "" {
		instances, err := p.LoadInstances()
		if err != nil {
			return nil, err
		}
		id = instances.GetDefault().GetID()
	} else {
		normalized, err := utils.NormalizeInstallURL(p.GetSiteURL(), explicit.String())
		if err != nil {
			return nil, err
		}
		id = types.ID(normalized)
	}

	instance, err := p.LoadInstance(id)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
