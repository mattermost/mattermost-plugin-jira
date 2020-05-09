// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
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

func NewInstances(initial ...*InstanceCommon) *Instances {
	instances := &Instances{
		ValueSet: types.NewValueSet(&instancesArray{}),
	}
	for _, ic := range initial {
		instances.Set(ic)
	}
	return instances
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

type instancesArray []*InstanceCommon

func (p instancesArray) Len() int                   { return len(p) }
func (p instancesArray) GetAt(n int) types.Value    { return p[n] }
func (p instancesArray) SetAt(n int, v types.Value) { p[n] = v.(*InstanceCommon) }

func (p instancesArray) InstanceOf() types.ValueArray {
	inst := make(instancesArray, 0)
	return &inst
}
func (p *instancesArray) Ref() interface{} { return &p }
func (p *instancesArray) Resize(n int) {
	*p = make(instancesArray, n)
}

func (p *Plugin) InstallInstance(instance Instance) error {
	err := p.instanceStore.UpdateInstances(
		func(instances *Instances) error {
			if instances.Contains(instance.Common().URL) {
				return ErrAlreadyExists
			}
			err := p.instanceStore.StoreInstance(instance)
			if err != nil {
				return err
			}
			instances.Set(instance.Common())
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
	err := p.instanceStore.UpdateInstances(
		func(instances *Instances) error {
			if !instances.Contains(id) {
				return ErrInstanceNotFound
			}
			var err error
			instance, err = p.instanceStore.LoadInstance(id)
			if err != nil {
				return err
			}
			if instanceType != instance.Common().Type {
				return errors.Errorf("%s did not match instance %s type %s", instanceType, id, instance.Common().Type)
			}
			return p.instanceStore.DeleteInstance(id)
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
	err := p.instanceStore.UpdateInstances(
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
		instances, err := p.instanceStore.LoadInstances()
		if err != nil {
			return nil, err
		}
		if instances.IsEmpty() {
			return nil, ErrInstanceNotFound
		}
		id = instances.GetDefault().GetID()
	} else {
		normalized, err := utils.NormalizeInstallURL(p.GetSiteURL(), explicit.String())
		if err != nil {
			return nil, err
		}
		id = types.ID(normalized)
	}

	instance, err := p.instanceStore.LoadInstance(id)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
