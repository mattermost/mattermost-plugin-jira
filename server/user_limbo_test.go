// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// disconnectUserStoreDouble is a small UserStore test double dedicated to
// the DisconnectUser tests below. It tracks DeleteConnection calls, which
// mockUserStoreKV does not, and its LoadConnection mirrors the real store's
// behavior of returning a non-nil, empty Connection (not an error) for a
// missing row.
type disconnectUserStoreDouble struct {
	mockUserStore
	users              map[types.ID]*User
	connections        map[types.ID]*Connection // keyed by mattermostUserID
	deletedConnections []types.ID               // instanceIDs passed to DeleteConnection
}

func (s *disconnectUserStoreDouble) LoadUser(id types.ID) (*User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, errors.Wrapf(kvstore.ErrNotFound, "user %q", id)
	}
	return u, nil
}

func (s *disconnectUserStoreDouble) StoreUser(user *User) error {
	s.users[user.MattermostUserID] = user
	return nil
}

func (s *disconnectUserStoreDouble) LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error) {
	c, ok := s.connections[mattermostUserID]
	if !ok {
		return &Connection{MattermostUserID: mattermostUserID}, nil
	}
	return c, nil
}

func (s *disconnectUserStoreDouble) DeleteConnection(instanceID, mattermostUserID types.ID) error {
	s.deletedConnections = append(s.deletedConnections, instanceID)
	delete(s.connections, mattermostUserID)
	return nil
}

func newPluginForDisconnectTests(t *testing.T, instances *Instances, userStore *disconnectUserStoreDouble) *Plugin {
	t.Helper()
	p := &Plugin{}
	api := &plugintest.API{}
	api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.tracker = &mockTelemetryTracker{}

	store := newInstanceStoreDouble()
	store.instances = instances
	p.instanceStore = store
	p.userStore = userStore
	return p
}

func TestDisconnectUser_UninstalledInstance(t *testing.T) {
	deadInstanceID := types.ID("https://dead-instance.example.com")
	userID := types.ID("test-user")

	user := NewUser(userID)
	user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
	user.DefaultInstanceID = deadInstanceID

	userStore := &disconnectUserStoreDouble{
		users: map[types.ID]*User{userID: user},
		connections: map[types.ID]*Connection{
			userID: {User: jira.User{AccountID: "dead-account", DisplayName: "Dead Account"}},
		},
	}
	p := newPluginForDisconnectTests(t, NewInstances(), userStore) // instances/v3 no longer lists deadInstanceID

	conn, err := p.DisconnectUser(deadInstanceID.String(), userID)
	require.NoError(t, err, "disconnecting from an uninstalled instance must succeed instead of requiring a live Instance")
	require.NotNil(t, conn)
	assert.Equal(t, "Dead Account", conn.DisplayName)

	updated, err := userStore.LoadUser(userID)
	require.NoError(t, err)
	assert.False(t, updated.ConnectedInstances.Contains(deadInstanceID), "the stale instance must be pruned from ConnectedInstances")
	assert.Empty(t, updated.DefaultInstanceID, "a default pointing at the disconnected instance must be cleared")
	assert.Contains(t, userStore.deletedConnections, deadInstanceID, "the connection row (and its reverse index) must be deleted")
}

func TestDisconnectUser_MissingConnectionRow(t *testing.T) {
	deadInstanceID := types.ID("https://dead-instance.example.com")
	userID := types.ID("test-user")

	user := NewUser(userID)
	user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
	user.DefaultInstanceID = deadInstanceID

	userStore := &disconnectUserStoreDouble{
		users:       map[types.ID]*User{userID: user},
		connections: map[types.ID]*Connection{}, // no connection row for userID
	}
	p := newPluginForDisconnectTests(t, NewInstances(), userStore)

	conn, err := p.DisconnectUser(deadInstanceID.String(), userID)
	require.NoError(t, err, "a missing connection row must not block cleanup of the user record")
	require.NotNil(t, conn, "executeDisconnect dereferences the returned Connection unconditionally")

	updated, err := userStore.LoadUser(userID)
	require.NoError(t, err)
	assert.False(t, updated.ConnectedInstances.Contains(deadInstanceID))
	assert.Empty(t, updated.DefaultInstanceID)
}
