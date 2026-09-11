// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const licenseErrorString = "You need a valid Mattermost Professional, Enterprise or Enterprise Advanced License to install multiple Jira instances."

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

// checkIfExists returns true if the specified instance ID already exists
func (instances Instances) checkIfExists(instanceID types.ID) bool {
	for _, id := range instances.IDs() {
		if id == instanceID {
			return true
		}
	}

	return false
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

func (p *Plugin) InstallInstance(newInstance Instance) error {
	var updated *Instances
	err := UpdateInstances(p.instanceStore,
		func(instances *Instances) error {
			if instances == nil {
				return errors.New("received nil 'instances' in UpdateInstances callback")
			}

			if !p.enterpriseChecker.HasEnterpriseFeatures() && len(instances.IDs()) > 0 && !instances.checkIfExists(newInstance.GetID()) {
				return errors.New(licenseErrorString)
			}

			err := p.instanceStore.StoreInstance(newInstance)
			if err != nil {
				return errors.Wrap(err, "failed to store new instance")
			}

			instances.Set(newInstance.Common())
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

func (p *Plugin) UninstallInstance(instanceID types.ID, instanceType InstanceType) (Instance, UninstallCleanup, error) {
	var instance Instance
	var updated *Instances
	var cleanup UninstallCleanup
	err := UpdateInstances(p.instanceStore,
		func(instances *Instances) error {
			if !instances.Contains(instanceID) {
				return errors.Wrapf(kvstore.ErrNotFound, "instance %q", instanceID)
			}

			var err error
			instance, err = p.instanceStore.LoadInstance(instanceID)
			switch {
			case err != nil && errors.Cause(err) != kvstore.ErrNotFound:
				return err

			case err != nil:
				// The instance blob is already gone, most likely because a
				// previous uninstall attempt deleted it but failed before
				// removing it from the list. Validate the type from the list
				// entry instead, and still run the user cleanup and list
				// removal below rather than leaving users pointed at a dead
				// instance.
				common := instances.Get(instanceID)
				if instanceType != common.Type {
					return errors.Errorf("%s did not match instance %s type %s", instanceType, instanceID, common.Type)
				}
				instance = stubInstance(common)

			default:
				if instanceType != instance.Common().Type {
					return errors.Errorf("%s did not match instance %s type %s", instanceType, instanceID, instance.Common().Type)
				}
			}

			cleanup, err = p.disconnectAllUsersFromInstance(instanceID)
			if err != nil {
				return err
			}

			instances.Delete(instanceID)
			updated = instances
			return nil
		})
	if err != nil {
		return nil, UninstallCleanup{}, err
	}

	// Delete the instance blob only after the instance list has been durably
	// updated to no longer reference it. Failing in the other order leaves a
	// list entry pointing at a deleted blob, which is the state that strands
	// users. A blob left behind with no list entry is the better failure:
	// instance enumeration and the user-record reconciliation both read the
	// list, so it does not strand anyone, and the setup handshake relies on
	// that same state for inactive Cloud instances anyway. Note it is not
	// fully inert -- LoadInstance resolves a blob by ID without consulting
	// the list -- so re-running the uninstall is what actually clears it.
	if err := p.instanceStore.DeleteInstance(instanceID); err != nil {
		p.errorf("UninstallInstance: failed to delete instance blob %q: %v", instanceID, err)
	}

	if updated == nil {
		updated = NewInstances()
	}

	// Re-register the /jira command with the new number of instances.
	if err := p.registerJiraCommand(p.getConfig().EnableAutocomplete, updated.Len() > 1); err != nil {
		p.errorf("UninstallInstance: failed to re-register `/%s` command; please re-activate the plugin using the System Console. Error: %s",
			commandTrigger, err.Error())
	}

	// Notify users we have uninstalled an instance
	p.wsInstancesChanged(updated)
	return instance, cleanup, nil
}

// UninstallCleanup reports what the post-uninstall user sweep could not
// finish. The two counts are not interchangeable: a failed disconnect names
// a user known to be connected to the removed instance, while an unreadable
// record may belong to a user who was never connected to it at all.
type UninstallCleanup struct {
	FailedDisconnects int
	UnreadableRecords int
}

// disconnectAllUsersFromInstance disconnects every user connected to
// instanceID. It does not require the instance to still be installed.
// Failures are logged and skipped rather than aborting the sweep, so that
// one bad record cannot leave the remaining users stuck.
func (p *Plugin) disconnectAllUsersFromInstance(instanceID types.ID) (UninstallCleanup, error) {
	cleanup := UninstallCleanup{}
	unreadable, err := p.userStore.MapUsers(func(user *User) error {
		if !user.ConnectedInstances.Contains(instanceID) {
			return nil
		}
		if _, err := p.disconnectUser(instanceID, user); err != nil {
			p.infof("UninstallInstance: failed to disconnect user %q from instance %q: %v", user.MattermostUserID, instanceID, err)
			cleanup.FailedDisconnects++
		}
		return nil
	})
	cleanup.UnreadableRecords = unreadable
	return cleanup, err
}

// stubInstance builds a minimal Instance from list metadata alone, for a
// record whose backing KV blob is already missing. It is only good for
// read-only accessors like GetManageAppsURL/GetManageWebhooksURL that
// callers need after a best-effort uninstall.
func stubInstance(common *InstanceCommon) Instance {
	switch common.Type {
	case CloudOAuthInstanceType:
		return &cloudOAuthInstance{InstanceCommon: common, JiraBaseURL: common.InstanceID.String()}
	case CloudInstanceType:
		return &cloudInstance{
			InstanceCommon:           common,
			AtlassianSecurityContext: &AtlassianSecurityContext{BaseURL: common.InstanceID.String()},
		}
	default:
		return &serverInstance{InstanceCommon: common}
	}
}

func (p *Plugin) wsInstancesChanged(instances *Instances) {
	msg := map[string]interface{}{
		"instances": instances.AsConfigMap(),
	}
	// Notify users we have uninstalled an instance
	p.client.Frontend.PublishWebSocketEvent(websocketEventInstanceStatus, msg, &model.WebsocketBroadcast{})
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
	if instanceURL != "" && isOpaqueCloudSetupRoutingID(instanceURL) {
		jiraURL, err := p.instanceStore.LoadPendingCloudSetupRoute(types.ID(instanceURL))
		if err != nil {
			return "", err
		}
		return jiraURL, nil
	}
	if instanceURL != "" {
		instanceURL, err = utils.NormalizeJiraURL(instanceURL)
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
		instanceURL, err = utils.NormalizeJiraURL(instanceURL)
		if err != nil {
			return "", err
		}
	}

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return "", errors.Wrap(err, "failed to load instances")
	}
	instance := instances.getByAlias(instanceURL)
	if instance != nil {
		instanceURL = instance.InstanceID.String()
	}

	// An explicitly requested instance is returned as-is, even if it is no
	// longer installed: callers such as /jira disconnect need to be able to
	// target a removed instance in order to clean up a stale record.
	if types.ID(instanceURL) != "" {
		return types.ID(instanceURL), nil
	}

	// Only consider instances the user is connected to that are still
	// installed when picking a default. A DefaultInstanceID or a lone
	// ConnectedInstances entry that points at a removed instance must not
	// be selected here, or every instance-resolving command fails as soon
	// as the referenced instance is gone.
	connected := NewInstances()
	for _, id := range user.ConnectedInstances.IDs() {
		if instances.Contains(id) {
			connected.Set(instances.Get(id))
		}
	}
	if connected.IsEmpty() {
		return "", errors.Wrap(kvstore.ErrNotFound, "your account is not connected to Jira. Please use `/jira connect`")
	}
	if user.DefaultInstanceID != "" && connected.Contains(user.DefaultInstanceID) {
		return user.DefaultInstanceID, nil
	}
	if connected.Len() == 1 {
		return connected.IDs()[0], nil
	}
	return "", errors.New("default jira instance not found, please run `/jira instance default <jiraURL>` to set one")
}

func (p *Plugin) httpAutocompleteConnect(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	info, err := p.GetUserInfo(mattermostUserID, nil)
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
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	info, err := p.GetUserInfo(mattermostUserID, nil)
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
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.Wrap(err, "failed to load instances"))
	}

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
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	info, err := p.GetUserInfo(mattermostUserID, nil)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	out := []model.AutocompleteListItem{}
	if info.User == nil {
		return respondJSON(w, out)
	}

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.Wrap(err, "failed to load instances"))
	}

	for _, instanceID := range info.Instances.IDs() {
		item := instances.getAlias(instanceID)
		helpText := string(instanceID)
		if item == "" {
			item = string(instanceID)
			helpText = ""
		}
		out = append(out, model.AutocompleteListItem{
			Item:     item,
			HelpText: helpText,
		})
	}
	return respondJSON(w, out)
}
func (p *Plugin) httpAutocompleteInstalledInstance(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := types.ID(r.Header.Get("Mattermost-User-Id"))
	info, err := p.GetUserInfo(mattermostUserID, nil)
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
