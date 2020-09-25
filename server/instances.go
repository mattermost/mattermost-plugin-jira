// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

type Instances struct {
	*types.ValueSet // of *InstanceCommon, not Instance
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

func (instances Instances) GetV2Legacy() *InstanceCommon {
	if instances.IsEmpty() {
		return nil
	}
	for _, id := range instances.ValueSet.IDs() {
		instance := instances.Get(id)
		if instance.IsV2Legacy {
			return instance
		}
	}
	return nil
}

func (instances Instances) SetV2Legacy(instanceID types.ID) error {
	if !instances.Contains(instanceID) {
		return errors.Wrapf(kvstore.ErrNotFound, "instance %q", instanceID)
	}

	prev := instances.GetV2Legacy()
	if prev != nil {
		prev.IsV2Legacy = false
	}
	instance := instances.Get(instanceID)
	instance.IsV2Legacy = true
	return nil
}

// getAlias returns the instance alias if it exists
func (instances Instances) getAlias(instanceID types.ID) string {
	for _, id := range instances.IDs() {
		instance := instances.Get(id)
		if instance.Common().InstanceID == instanceID {
			return instance.Common().Alias
		}
	}
	return ""
}

// getByAlias returns an instance with the requested alias
func (instances Instances) getByAlias(alias string) (instance *InstanceCommon) {
	if alias == "" {
		return nil
	}
	for _, id := range instances.IDs() {
		instance := instances.Get(id)
		if instance.Common().Alias == alias {
			return instance
		}
	}
	return nil
}

// isAliasUnique returns true if no other instance has the requested alias
func (instances Instances) isAliasUnique(instanceID types.ID, alias string) (bool, types.ID) {
	for _, id := range instances.IDs() {
		instance := instances.Get(id)
		if instance.Common().Alias == alias && instance.Common().InstanceID != instanceID {
			return false, instance.Common().InstanceID
		}
	}

	return true, ""
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
	err := UpdateInstances(p.instanceStore,
		func(instances *Instances) error {
			if !p.enterpriseChecker.HasEnterpriseFeatures() {
				if instances != nil && len(instances.IDs()) > 0 {
					return errors.Errorf("You need an Enterprise License to install multiple Jira instances")
				}
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

	// Re-register the /jira command with the new number of instances.
	err = p.registerJiraCommand(p.getConfig().EnableAutocomplete, updated.Len() > 1)
	if err != nil {
		p.errorf("InstallInstance: failed to re-register `/%s` command; please re-activate the plugin using the System Console. Error: %s",
			commandTrigger, err.Error())
	}
	p.wsInstancesChanged(updated)
	return nil
}

func (p *Plugin) UninstallInstance(instanceID types.ID, instanceType InstanceType) (Instance, error) {
	var instance Instance
	var updated *Instances
	err := UpdateInstances(p.instanceStore,
		func(instances *Instances) error {
			if !instances.Contains(instanceID) {
				return errors.Wrapf(kvstore.ErrNotFound, "instance %q", instanceID)
			}
			var err error
			instance, err = p.instanceStore.LoadInstance(instanceID)
			if err != nil {
				return err
			}
			if instanceType != instance.Common().Type {
				return errors.Errorf("%s did not match instance %s type %s", instanceType, instanceID, instance.Common().Type)
			}

			p.userStore.MapUsers(func(user *User) error {
				if !user.ConnectedInstances.Contains(instance.GetID()) {
					return nil
				}

				_, err = p.disconnectUser(instance, user)
				if err != nil {
					p.infof("UninstallInstance: failed to disconnect user: %v", err)
				}
				return nil
			})

			instances.Delete(instanceID)
			updated = instances
			return p.instanceStore.DeleteInstance(instanceID)
		})
	if err != nil {
		return nil, err
	}

	// Re-register the /jira command with the new number of instances.
	err = p.registerJiraCommand(p.getConfig().EnableAutocomplete, updated.Len() > 1)
	if err != nil {
		p.errorf("UninstallInstance: failed to re-register `/%s` command; please re-activate the plugin using the System Console. Error: %s",
			commandTrigger, err.Error())
	}

	// Notify users we have uninstalled an instance
	p.wsInstancesChanged(updated)
	return instance, nil
}

func (p *Plugin) wsInstancesChanged(instances *Instances) {
	msg := map[string]interface{}{
		"instances": instances.AsConfigMap(),
	}
	// Notify users we have uninstalled an instance
	p.API.PublishWebSocketEvent(websocketEventInstanceStatus, msg, &model.WebsocketBroadcast{})
}

func (p *Plugin) StoreV2LegacyInstance(id types.ID) error {
	err := UpdateInstances(p.instanceStore,
		func(instances *Instances) error {
			return instances.SetV2Legacy(id)
		})
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) ResolveWebhookInstanceURL(instanceURL string) (types.ID, error) {
	var err error
	if instanceURL != "" {
		instanceURL, err = utils.NormalizeInstallURL(p.GetSiteURL(), instanceURL)
		if err != nil {
			return "", err
		}
	}
	instanceID := types.ID(instanceURL)
	if instanceID == "" {
		instances, err := p.instanceStore.LoadInstances()
		if err != nil {
			return "", err
		}
		if instances.IsEmpty() {
			return "", errors.Wrap(kvstore.ErrNotFound, "no instances installed")
		}
		v2 := instances.GetV2Legacy()
		switch {
		case v2 != nil:
			instanceID = v2.InstanceID
		case instances.Len() == 1:
			instanceID = instances.IDs()[0]
		default:
			return "", errors.Wrap(kvstore.ErrNotFound, "specify a Jira instance")
		}
	}
	return instanceID, nil
}

func (p *Plugin) LoadUserInstance(mattermostUserID types.ID, instanceURL string) (*User, Instance, error) {
	user, instanceID, err := p.ResolveUserInstanceURL(mattermostUserID, instanceURL)
	if err != nil {
		return nil, nil, err
	}

	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return nil, nil, err
	}
	return user, instance, nil
}

func (p *Plugin) ResolveUserInstanceURL(mattermostUserID types.ID, instanceURL string) (*User, types.ID, error) {
	user, err := p.userStore.LoadUser(mattermostUserID)
	if err != nil {
		return nil, "", err
	}
	instanceID, err := p.resolveUserInstanceURL(user, instanceURL)
	if err != nil {
		return nil, "", err
	}
	return user, instanceID, nil
}

func (p *Plugin) resolveUserInstanceURL(user *User, instanceURL string) (types.ID, error) {
	if user.ConnectedInstances.IsEmpty() {
		return "", errors.Wrap(kvstore.ErrNotFound, "your account is not connected to Jira. Please use `/jira connect`")
	}

	var err error
	if instanceURL != "" {
		instanceURL, err = utils.NormalizeInstallURL(p.GetSiteURL(), instanceURL)
		if err != nil {
			return "", err
		}
	}

	instances, err := p.instanceStore.LoadInstances()
	instance := instances.getByAlias(instanceURL)
	if instance != nil {
		instanceURL = instance.InstanceID.String()
	}

	if types.ID(instanceURL) != "" {
		return types.ID(instanceURL), nil
	}
	if user.DefaultInstanceID != "" && user.ConnectedInstances.Contains(user.DefaultInstanceID) {
		return user.DefaultInstanceID, nil
	}
	if user.ConnectedInstances.Len() == 1 {
		return user.ConnectedInstances.IDs()[0], nil
	}

	return "", errors.Wrap(kvstore.ErrNotFound, "unable to pick the default Jira instance")
}

func (p *Plugin) httpAutocompleteConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	info, err := p.GetUserInfo(mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	out := []model.AutocompleteListItem{}
	for _, instanceID := range info.connectable.IDs() {
		out = append(out, model.AutocompleteListItem{
			Item: instanceID.String(),
		})
	}
	return respondJSON(w, out)
}

func (p *Plugin) httpAutocompleteUserInstance(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	info, err := p.GetUserInfo(mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	out := []model.AutocompleteListItem{}
	if info.User == nil {
		return respondJSON(w, out)
	}

	// Put the default in first
	if info.User.DefaultInstanceID != "" {
		out = append(out, model.AutocompleteListItem{
			Item: info.User.DefaultInstanceID.String(),
		})
	}

	instances, err := p.instanceStore.LoadInstances()
	for _, instanceID := range info.User.ConnectedInstances.IDs() {
		if instanceID != info.User.DefaultInstanceID {
			id := instances.getAlias(instanceID)
			if id == "" {
				id = instanceID.String()
			}
			out = append(out, model.AutocompleteListItem{
				Item: id,
			})
		}
	}
	return respondJSON(w, out)
}

func (p *Plugin) httpAutocompleteInstalledInstanceWithAlias(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	info, err := p.GetUserInfo(mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	out := []model.AutocompleteListItem{}
	if info.User == nil {
		return respondJSON(w, out)
	}

	instances, err := p.instanceStore.LoadInstances()
	for _, instanceID := range info.Instances.IDs() {
		alias := instances.getAlias(instanceID)
		if alias != "" {
			out = append(out, model.AutocompleteListItem{
				Item: alias,
			})
		}
	}
	return respondJSON(w, out)
}
func (p *Plugin) httpAutocompleteInstalledInstance(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	info, err := p.GetUserInfo(mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	out := []model.AutocompleteListItem{}
	if info.User == nil {
		return respondJSON(w, out)
	}

	for _, instanceID := range info.Instances.IDs() {
		out = append(out, model.AutocompleteListItem{
			Item: instanceID.String(),
		})
	}
	return respondJSON(w, out)
}
