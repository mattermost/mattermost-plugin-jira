// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"path/filepath"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// instanceStoreDouble is a minimal InstanceStore test double that mirrors
// the real store's kvstore.ErrNotFound wrapping for a missing instance
// blob. mockInstanceStoreKV (used by other tests) returns a bare error
// instead, which several other tests assert on verbatim, so this is kept
// separate rather than changing that shared double.
type instanceStoreDouble struct {
	mockInstanceStore
	instances *Instances
	blobs     map[types.ID]Instance
	// order records the sequence of StoreInstances/DeleteInstance calls, to
	// assert on write ordering.
	order []string
}

func newInstanceStoreDouble() *instanceStoreDouble {
	return &instanceStoreDouble{
		instances: NewInstances(),
		blobs:     map[types.ID]Instance{},
	}
}

func (s *instanceStoreDouble) LoadInstances() (*Instances, error) {
	return s.instances, nil
}

func (s *instanceStoreDouble) StoreInstances(instances *Instances) error {
	s.instances = instances
	s.order = append(s.order, "StoreInstances")
	return nil
}

func (s *instanceStoreDouble) LoadInstance(id types.ID) (Instance, error) {
	instance, ok := s.blobs[id]
	if !ok {
		return nil, errors.Wrap(kvstore.ErrNotFound, string(id))
	}
	return instance, nil
}

func (s *instanceStoreDouble) StoreInstance(instance Instance) error {
	s.blobs[instance.GetID()] = instance
	return nil
}

func (s *instanceStoreDouble) DeleteInstance(id types.ID) error {
	delete(s.blobs, id)
	s.order = append(s.order, "DeleteInstance")
	return nil
}

func newPluginForInstanceTests(t *testing.T, instanceStore InstanceStore) *Plugin {
	t.Helper()
	p := &Plugin{}
	api := &plugintest.API{}
	bundlePath, err := filepath.Abs("..")
	require.NoError(t, err)
	api.On("GetBundlePath").Return(bundlePath, nil).Maybe()
	api.On("GetConfig").Return(&model.Config{}).Maybe()
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("RegisterCommand", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	// p.infof/debugf/errorf (used on best-effort disconnect failures) log a
	// single pre-formatted message with no key-value pairs.
	api.On("LogDebug", mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything).Maybe()
	api.On("LogError", mock.Anything).Maybe()
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.instanceStore = instanceStore
	p.tracker = &mockTelemetryTracker{}
	return p
}

func TestResolveUserInstanceURL_StaleDefault(t *testing.T) {
	store := newInstanceStoreDouble()
	store.instances.Set(testInstance1.Common())
	store.blobs[testInstance1.InstanceID] = testInstance1

	p := newPluginForInstanceTests(t, store)

	deadInstanceID := types.ID("https://dead-instance.example.com")

	t.Run("default points at an uninstalled instance, resolves to the remaining installed one", func(t *testing.T) {
		user := NewUser("test-user")
		user.ConnectedInstances.Set(testInstance1.Common())
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
		user.DefaultInstanceID = deadInstanceID

		instanceID, err := p.resolveUserInstanceURL(user, "")
		require.NoError(t, err)
		assert.Equal(t, testInstance1.InstanceID, instanceID)
	})

	t.Run("only connected to an uninstalled instance resolves to not-connected, not a per-instance not-found", func(t *testing.T) {
		user := NewUser("test-user")
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
		user.DefaultInstanceID = deadInstanceID

		_, err := p.resolveUserInstanceURL(user, "")
		require.Error(t, err)
		assert.Equal(t, kvstore.ErrNotFound, errors.Cause(err))
	})

	t.Run("an explicit instance URL is still honored even when it is no longer installed", func(t *testing.T) {
		user := NewUser("test-user")
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})

		instanceID, err := p.resolveUserInstanceURL(user, deadInstanceID.String())
		require.NoError(t, err)
		assert.Equal(t, deadInstanceID, instanceID)
	})
}

func TestUninstallInstance_MissingBlob(t *testing.T) {
	deadInstanceID := types.ID("https://dead-instance.example.com")

	store := newInstanceStoreDouble()
	store.instances.Set(&InstanceCommon{InstanceID: deadInstanceID, Type: ServerInstanceType})
	// Note: no blob stored for deadInstanceID, simulating a previous
	// uninstall attempt that deleted the blob but failed before removing
	// the instance from the list.

	p := newPluginForInstanceTests(t, store)

	affectedUser := NewUser("affected-user")
	affectedUser.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
	affectedUser.DefaultInstanceID = deadInstanceID
	unaffectedUser := NewUser("unaffected-user")
	unaffectedUser.ConnectedInstances.Set(testInstance1.Common())

	p.userStore = mockUserStoreKV{
		users: map[types.ID]*User{
			affectedUser.MattermostUserID:   affectedUser,
			unaffectedUser.MattermostUserID: unaffectedUser,
		},
		connections: map[types.ID]*Connection{
			affectedUser.MattermostUserID: {User: jira.User{AccountID: "dead-account"}},
		},
	}

	instance, failedUsers, err := p.UninstallInstance(deadInstanceID, ServerInstanceType)
	require.NoError(t, err, "must not error, and must not panic, on a missing instance blob")
	require.NotNil(t, instance, "must return a usable stand-in Instance so callers can print manage-app URLs")
	assert.Equal(t, 0, failedUsers)

	instances, err := p.instanceStore.LoadInstances()
	require.NoError(t, err)
	assert.False(t, instances.Contains(deadInstanceID), "the dead instance must be removed from instances/v3")

	updated, err := p.userStore.LoadUser(affectedUser.MattermostUserID)
	require.NoError(t, err)
	assert.False(t, updated.ConnectedInstances.Contains(deadInstanceID), "the affected user's stale reference must be cleaned up")
	assert.Empty(t, updated.DefaultInstanceID)

	stillFine, err := p.userStore.LoadUser(unaffectedUser.MattermostUserID)
	require.NoError(t, err)
	assert.True(t, stillFine.ConnectedInstances.Contains(testInstance1.InstanceID), "an unrelated user's record must be untouched")
}

func TestUninstallInstance_WriteOrdering(t *testing.T) {
	liveInstanceID := testInstance1.InstanceID

	store := newInstanceStoreDouble()
	store.instances.Set(testInstance1.Common())
	store.blobs[liveInstanceID] = testInstance1

	p := newPluginForInstanceTests(t, store)
	p.userStore = mockUserStoreKV{users: map[types.ID]*User{}, connections: map[types.ID]*Connection{}}

	_, _, err := p.UninstallInstance(liveInstanceID, testInstance1.Type)
	require.NoError(t, err)

	require.Equal(t, []string{"StoreInstances", "DeleteInstance"}, store.order,
		"instances/v3 must be durably updated before the instance blob is deleted, "+
			"so a failure in between cannot leave the list pointing at a dead instance")
}
