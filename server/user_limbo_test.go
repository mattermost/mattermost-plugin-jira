// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// connKey identifies a connection row the way the real store does, by
// instance as well as by user, so tests can tell a removed instance's row
// apart from a live one belonging to the same user.
type connKey struct {
	instanceID       types.ID
	mattermostUserID types.ID
}

// limboUserStore is a UserStore test double for the stale-instance tests
// below. It tracks DeleteConnection calls, which mockUserStoreKV does not,
// and its LoadConnection mirrors the real store's behavior of returning a
// non-nil, empty Connection (not an error) for a missing row.
type limboUserStore struct {
	mockUserStore
	users              map[types.ID]*User
	connections        map[connKey]*Connection
	deletedConnections []connKey
}

func newLimboUserStore(users ...*User) *limboUserStore {
	s := &limboUserStore{
		users:       map[types.ID]*User{},
		connections: map[connKey]*Connection{},
	}
	for _, u := range users {
		s.users[u.MattermostUserID] = u
	}
	return s
}

func (s *limboUserStore) setConnection(instanceID, mattermostUserID types.ID, accountID, displayName string) {
	s.connections[connKey{instanceID, mattermostUserID}] = &Connection{
		User: jira.User{AccountID: accountID, DisplayName: displayName},
	}
}

func (s *limboUserStore) LoadUser(id types.ID) (*User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, errors.Wrapf(kvstore.ErrNotFound, "user %q", id)
	}
	return u, nil
}

func (s *limboUserStore) StoreUser(user *User) error {
	s.users[user.MattermostUserID] = user
	return nil
}

func (s *limboUserStore) LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error) {
	c, ok := s.connections[connKey{instanceID, mattermostUserID}]
	if !ok {
		return &Connection{MattermostUserID: mattermostUserID}, nil
	}
	return c, nil
}

func (s *limboUserStore) DeleteConnection(instanceID, mattermostUserID types.ID) error {
	key := connKey{instanceID, mattermostUserID}
	s.deletedConnections = append(s.deletedConnections, key)
	delete(s.connections, key)
	return nil
}

func newPluginForLimboTests(t *testing.T, instances *Instances, userStore *limboUserStore) *Plugin {
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

	userStore := newLimboUserStore(user)
	userStore.setConnection(deadInstanceID, userID, "dead-account", "Dead Account")

	p := newPluginForLimboTests(t, NewInstances(), userStore) // instances/v3 no longer lists deadInstanceID

	conn, err := p.DisconnectUser(deadInstanceID.String(), userID)
	require.NoError(t, err, "disconnecting from an uninstalled instance must succeed instead of requiring a live Instance")
	require.NotNil(t, conn)
	assert.Equal(t, "Dead Account", conn.DisplayName)

	updated, err := userStore.LoadUser(userID)
	require.NoError(t, err)
	assert.False(t, updated.ConnectedInstances.Contains(deadInstanceID), "the stale instance must be pruned from ConnectedInstances")
	assert.Empty(t, updated.DefaultInstanceID, "a default pointing at the disconnected instance must be cleared")
	assert.Contains(t, userStore.deletedConnections, connKey{deadInstanceID, userID}, "the connection row (and its reverse index) must be deleted")
}

func TestDisconnectUser_MissingConnectionRow(t *testing.T) {
	deadInstanceID := types.ID("https://dead-instance.example.com")
	userID := types.ID("test-user")

	user := NewUser(userID)
	user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
	user.DefaultInstanceID = deadInstanceID

	userStore := newLimboUserStore(user) // no connection row for userID
	p := newPluginForLimboTests(t, NewInstances(), userStore)

	conn, err := p.DisconnectUser(deadInstanceID.String(), userID)
	require.NoError(t, err, "a missing connection row must not block cleanup of the user record")
	require.NotNil(t, conn, "executeDisconnect dereferences the returned Connection unconditionally")

	updated, err := userStore.LoadUser(userID)
	require.NoError(t, err)
	assert.False(t, updated.ConnectedInstances.Contains(deadInstanceID))
	assert.Empty(t, updated.DefaultInstanceID)
}

// TestDisconnectUser_ClearsOtherDanglingInstances covers a record that
// accumulated more than one reference to a removed instance. Disconnecting
// from one of them must take the rest with it, rows included, rather than
// leaving the user to guess the URL of every instance that ever went away.
func TestDisconnectUser_ClearsOtherDanglingInstances(t *testing.T) {
	firstDead := types.ID("https://dead-one.example.com")
	secondDead := types.ID("https://dead-two.example.com")
	userID := types.ID("test-user")

	user := NewUser(userID)
	user.ConnectedInstances.Set(&InstanceCommon{InstanceID: firstDead})
	user.ConnectedInstances.Set(&InstanceCommon{InstanceID: secondDead})
	user.DefaultInstanceID = secondDead

	userStore := newLimboUserStore(user)
	userStore.setConnection(firstDead, userID, "first-account", "First Account")
	userStore.setConnection(secondDead, userID, "second-account", "Second Account")

	p := newPluginForLimboTests(t, NewInstances(), userStore)

	_, err := p.DisconnectUser(firstDead.String(), userID)
	require.NoError(t, err)

	updated, err := userStore.LoadUser(userID)
	require.NoError(t, err)
	assert.True(t, updated.ConnectedInstances.IsEmpty(), "every dangling reference must be cleared, not just the one named")
	assert.Empty(t, updated.DefaultInstanceID)
	assert.ElementsMatch(t,
		[]connKey{{firstDead, userID}, {secondDead, userID}},
		userStore.deletedConnections,
		"both orphaned rows must go, or they keep blocking a reconnect at the same URL")
}

// TestHealUserRecord covers the path that unsticks users without any
// command: GetUserInfo reconciles the record, and healUserRecord persists
// that and clears what the removed instances left in the KV store. The
// connection row matters most -- it is what makes a reinstall at the same
// URL look like an existing link.
func TestHealUserRecord(t *testing.T) {
	deadInstanceID := types.ID("https://dead-instance.example.com")
	userID := types.ID("test-user")

	user := NewUser(userID)
	user.ConnectedInstances.Set(testInstance1.Common())
	user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
	user.DefaultInstanceID = deadInstanceID

	userStore := newLimboUserStore(user)
	userStore.setConnection(testInstance1.InstanceID, userID, "live-account", "Live Account")
	userStore.setConnection(deadInstanceID, userID, "dead-account", "Dead Account")

	p := newPluginForLimboTests(t, NewInstances(testInstance1.Common()), userStore)

	info, err := p.GetUserInfo(userID, user)
	require.NoError(t, err)
	require.True(t, info.reconciled, "a record referencing a removed instance must be reported as needing a heal")
	require.Equal(t, []types.ID{deadInstanceID}, info.staleInstances)

	require.NoError(t, p.healUserRecord(info))

	updated, err := userStore.LoadUser(userID)
	require.NoError(t, err)
	assert.True(t, updated.ConnectedInstances.Contains(testInstance1.InstanceID), "the still-installed instance must be kept")
	assert.False(t, updated.ConnectedInstances.Contains(deadInstanceID))
	assert.Empty(t, updated.DefaultInstanceID, "a default pointing at the removed instance must be cleared")

	assert.Equal(t, []connKey{{deadInstanceID, userID}}, userStore.deletedConnections,
		"only the removed instance's row may be deleted; the live connection must survive")
}

func TestHealUserRecord_NothingToDo(t *testing.T) {
	userID := types.ID("test-user")

	user := NewUser(userID)
	user.ConnectedInstances.Set(testInstance1.Common())
	user.DefaultInstanceID = testInstance1.InstanceID

	userStore := newLimboUserStore(user)
	userStore.setConnection(testInstance1.InstanceID, userID, "live-account", "Live Account")

	p := newPluginForLimboTests(t, NewInstances(testInstance1.Common()), userStore)

	info, err := p.GetUserInfo(userID, user)
	require.NoError(t, err)
	require.False(t, info.reconciled)

	require.NoError(t, p.healUserRecord(info))
	assert.Empty(t, userStore.deletedConnections, "a healthy record must not lose its connection rows")
	assert.Equal(t, testInstance1.InstanceID, user.DefaultInstanceID)
}

// TestConnectRoutesWithOrphanedConnectionRow covers the limbo loop where a
// connection row outlives the instance that wrote it, e.g. an instance
// removed and then reinstalled at the same URL. Only the user's own record
// may decide they are already linked; a row that record does not back must
// be cleared and the connect flow allowed to proceed, instead of answering
// with a 400 telling the user to disconnect from something the disconnect
// path does not consider them connected to.
func TestConnectRoutesWithOrphanedConnectionRow(t *testing.T) {
	const (
		orphanedRowUserID = "orphaned_row_user"
		connectedUserID   = "connected_user"
	)

	connectRoute := instancePath(routeUserConnect, testInstance1.InstanceID)

	for name, tc := range map[string]struct {
		route              string
		userID             string
		expectedStatusCode int
		expectRowDeleted   bool
	}{
		"connect proceeds for an orphaned row": {
			route:              connectRoute,
			userID:             orphanedRowUserID,
			expectedStatusCode: http.StatusFound,
			expectRowDeleted:   true,
		},
		"start attempts the connect for an orphaned row": {
			route:              routeUserStart,
			userID:             orphanedRowUserID,
			expectedStatusCode: http.StatusFound,
			expectRowDeleted:   true,
		},
		"connect still reports an existing link for a genuinely connected user": {
			route:              connectRoute,
			userID:             connectedUserID,
			expectedStatusCode: http.StatusBadRequest,
			expectRowDeleted:   false,
		},
		"start still redirects a genuinely connected user to the docs": {
			route:              routeUserStart,
			userID:             connectedUserID,
			expectedStatusCode: http.StatusSeeOther,
			expectRowDeleted:   false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 13)...).Return().Maybe()
			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return().Maybe()

			p := &Plugin{}
			p.initializeRouter()
			p.SetAPI(api)

			store := getMockUserStoreKV()
			// A record listing no instances at all, but with a connection
			// row still sitting in the KV store.
			store.users[orphanedRowUserID] = NewUser(orphanedRowUserID)
			store.connections[orphanedRowUserID] = &Connection{User: jira.User{AccountID: "stale-account"}}
			p.userStore = store
			p.instanceStore = p.getMockInstanceStoreKV(1)

			request := httptest.NewRequest(http.MethodGet, tc.route, nil)
			request.Header.Set(headerMattermostUserID, tc.userID)
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)

			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)

			_, rowStillThere := store.connections[types.ID(tc.userID)]
			assert.Equal(t, tc.expectRowDeleted, !rowStillThere,
				"an orphaned row must be cleared, and a row backed by the user's record must be left alone")
		})
	}
}
