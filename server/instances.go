// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

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

func (instances Instances) Set(ic *InstanceCommon) {
	instances.ValueSet.Set(ic)
}

func (instances Instances) AsConfigMap() []interface{} {
	out := []interface{}{}
	for _, id := range instances.IDs() {
		instance := instances.Get(id)
		out = append(out, instance.Common().AsConfigMap())
	}

	return out
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
	var updated *Instances
	err := p.instanceStore.UpdateInstances(
		func(instances *Instances) error {
			if instances.Contains(instance.GetID()) {
				return ErrAlreadyExists
			}
			err := p.instanceStore.StoreInstance(instance)
			if err != nil {
				return err
			}
			instances.Set(instance.Common())
			updated = instances
			return nil
		})
	if err != nil {
		return err
	}

	// Notify users we have installed an instance
	p.API.PublishWebSocketEvent(websocketEventInstanceStatus,
		map[string]interface{}{
			"instances": updated.AsConfigMap(),
		},
		&model.WebsocketBroadcast{})
	return nil
}

var ErrInstanceNotFound = errors.New("instance not found")

func (p *Plugin) UninstallInstance(id types.ID, instanceType InstanceType) (Instance, error) {
	var instance Instance
	var updated *Instances
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
			updated = instances
			return p.instanceStore.DeleteInstance(id)
		})
	if err != nil {
		return nil, err
	}

	// Notify users we have uninstalled an instance
	p.API.PublishWebSocketEvent(websocketEventInstanceStatus,
		map[string]interface{}{
			"instances": updated.AsConfigMap(),
		},
		&model.WebsocketBroadcast{})
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
	id, err := p.ResolveInstanceID(explicit)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.New("No instance available")
	}
	instance, err := p.instanceStore.LoadInstance(id)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (p *Plugin) ResolveInstanceID(explicit types.ID) (types.ID, error) {
	if explicit != "" {
		return explicit, nil
	}

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return "", err
	}
	if instances.IsEmpty() {
		return "", ErrInstanceNotFound
	}
	return instances.GetDefault().GetID(), nil
}

func (p *Plugin) ResolveInstanceURL(jiraurl string) (types.ID, error) {
	if jiraurl != "" {
		var err error
		jiraurl, err = utils.NormalizeInstallURL(p.GetSiteURL(), jiraurl)
		if err != nil {
			return "", err
		}
		return types.ID(jiraurl), err
	}
	return p.ResolveInstanceID(types.ID(jiraurl))
}
