// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func TestUserSettings_String(t *testing.T) {
	tests := map[string]struct {
		settings       ConnectionSettings
		expectedOutput string
	}{
		"notifications on": {
			settings: ConnectionSettings{
				Notifications: true,
				RolesForDMNotification: map[string]bool{
					assigneeRole: true,
					mentionRole:  true,
					reporterRole: true,
					watchingRole: true,
				},
			},
			expectedOutput: "\t- Notifications for assignee: on \n\t- Notifications for mention: on \n\t- Notifications for reporter: on \n\t- Notifications for watching: on",
		},
		"notifications off": {
			settings: ConnectionSettings{
				Notifications: false,
				RolesForDMNotification: map[string]bool{
					assigneeRole: false,
					mentionRole:  false,
					reporterRole: false,
					watchingRole: false,
				},
			},
			expectedOutput: "\t- Notifications for assignee: off \n\t- Notifications for mention: off \n\t- Notifications for reporter: off \n\t- Notifications for watching: off",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOutput, tt.settings.String())
		})
	}
}

func TestRouteUserConnectAndStart(t *testing.T) {
	// A record listing no instances at all, but with a connection row left
	// in the KV store, as a since-removed instance at this URL leaves
	// behind. Only the record may decide the user is already linked, so the
	// row has to be cleared rather than block the connect flow.
	const orphanedRowUserID = "orphaned_row_user"

	tests := map[string]struct {
		route      string
		userID     string
		statusCode int
		// expectRowKept is only checked for the cases whose fixture starts
		// with a connection row.
		expectRowKept bool
	}{
		"user connected to jira will re-direct to docs": {
			route: routeUserStart, userID: "connected_user", statusCode: http.StatusSeeOther, expectRowKept: true,
		},
		"user not connected to jira will atempt connect": {
			route: routeUserStart, userID: "non_connected_user", statusCode: http.StatusFound,
		},
		"user with an orphaned connection row will attempt connect": {
			route:      instancePath(routeUserConnect, testInstance1.InstanceID),
			userID:     orphanedRowUserID,
			statusCode: http.StatusFound,
		},
		"user whose record backs the connection row is told to disconnect": {
			route: instancePath(routeUserConnect, testInstance1.InstanceID), userID: "connected_user",
			statusCode: http.StatusBadRequest, expectRowKept: true,
		},
	}
	api := &plugintest.API{}

	api.On("LogWarn", mockAnythingOfTypeBatch("string", 13)...).Return()

	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return()

	p := Plugin{}
	p.initializeRouter()
	p.SetAPI(api)

	store := getMockUserStoreKV()
	store.users[orphanedRowUserID] = NewUser(orphanedRowUserID)
	store.connections[orphanedRowUserID] = &Connection{User: jira.User{AccountID: "stale-account"}}
	p.userStore = store
	p.instanceStore = p.getMockInstanceStoreKV(1)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest("GET", tc.route, nil)
			request.Header.Set("Mattermost-User-Id", tc.userID)
			_, hadRow := store.connections[types.ID(tc.userID)]

			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tc.statusCode, w.Result().StatusCode)

			if hadRow {
				_, kept := store.connections[types.ID(tc.userID)]
				assert.Equal(t, tc.expectRowKept, kept,
					"an orphaned row must be cleared, and a row the user's record backs must be left alone")
			}
		})
	}
}

// TestDisconnectUserFromUninstalledInstance covers the reported limbo
// state, where a record pointing at a removed instance left the user unable
// to either use another instance or disconnect from the dead one. The
// disconnect must now succeed and take every dangling reference with it,
// connection rows included, while leaving a live connection alone.
func TestDisconnectUserFromUninstalledInstance(t *testing.T) {
	const userID = types.ID("test-user")
	firstDead := types.ID("https://dead-one.example.com")
	secondDead := types.ID("https://dead-two.example.com")

	setup := func(withConnectionRows bool) (*Plugin, *limboUserStore) {
		user := NewUser(userID)
		user.ConnectedInstances.Set(testInstance1.Common())
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: firstDead})
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: secondDead})
		user.DefaultInstanceID = secondDead

		store := &limboUserStore{
			users:       map[types.ID]*User{userID: user},
			connections: map[connKey]*Connection{},
		}
		if withConnectionRows {
			store.connections[connKey{testInstance1.InstanceID, userID}] = &Connection{User: jira.User{AccountID: "live-account"}}
			store.connections[connKey{firstDead, userID}] = &Connection{User: jira.User{AccountID: "first-account", DisplayName: "First Account"}}
			store.connections[connKey{secondDead, userID}] = &Connection{User: jira.User{AccountID: "second-account"}}
		}

		// instances/v3 lists only testInstance1; both dead instances are gone.
		p := newPluginForStoreTests(t, newInstanceStoreDouble(testInstance1))
		p.userStore = store
		return p, store
	}

	t.Run("clears every dangling reference and its connection row", func(t *testing.T) {
		p, store := setup(true)

		conn, err := p.DisconnectUser(firstDead.String(), userID)
		require.NoError(t, err, "a removed instance must still be disconnectable")
		assert.Equal(t, "First Account", conn.DisplayName)

		updated, err := store.LoadUser(userID)
		require.NoError(t, err)
		assert.True(t, updated.ConnectedInstances.Contains(testInstance1.InstanceID), "the installed instance must be kept")
		assert.False(t, updated.ConnectedInstances.Contains(firstDead))
		assert.False(t, updated.ConnectedInstances.Contains(secondDead), "one disconnect must clear every dangling reference, not just the one named")
		assert.Empty(t, updated.DefaultInstanceID)

		assert.ElementsMatch(t,
			[]connKey{{firstDead, userID}, {secondDead, userID}},
			store.deletedConnections,
			"both orphaned rows must go, or they keep blocking a reconnect at the same URL, and the live one must survive")
	})

	t.Run("succeeds when the connection row is already gone", func(t *testing.T) {
		p, store := setup(false)

		conn, err := p.DisconnectUser(firstDead.String(), userID)
		require.NoError(t, err, "a missing row must not block cleanup of the record")
		require.NotNil(t, conn, "executeDisconnect dereferences the returned Connection unconditionally")

		updated, err := store.LoadUser(userID)
		require.NoError(t, err)
		assert.False(t, updated.ConnectedInstances.Contains(firstDead))
		assert.Empty(t, updated.DefaultInstanceID)
	})

	t.Run("reports success when only the opportunistic heal fails", func(t *testing.T) {
		p, store := setup(true)
		// The requested disconnect is written first; fail the second write,
		// which is the bonus cleanup of the other dead instance.
		store.storeUserErrAfter = 1

		conn, err := p.DisconnectUser(firstDead.String(), userID)
		require.NoError(t, err, "the requested disconnect is already persisted, so a failed heal must not be reported as a failed disconnect")
		require.NotNil(t, conn)

		updated, err := store.LoadUser(userID)
		require.NoError(t, err)
		assert.False(t, updated.ConnectedInstances.Contains(firstDead), "the requested disconnect must stand")

		p.API.(*plugintest.API).AssertCalled(t, "PublishWebSocketEvent",
			websocketEventDisconnect, mock.Anything, mock.Anything)
	})
}

func TestGetJiraUserFromMentions(t *testing.T) {
	p := Plugin{}
	p.userStore = getMockUserStoreKV()
	p.instanceStore = p.getMockInstanceStoreKV(1)
	testUser, err := p.userStore.LoadUser("connected_user")
	assert.Nil(t, err)

	tests := map[string]struct {
		mentions       *model.UserMentionMap
		userSearch     string
		expectedResult *jira.User
		expectedError  string
		SetupAPI       func(api *plugintest.API)
	}{
		"if no mentions, no users are returned": {
			mentions:      &model.UserMentionMap{},
			userSearch:    "join",
			expectedError: "the mentioned user was not found",
			SetupAPI:      func(api *plugintest.API) {},
		},
		"non connected user won't appear when mentioned": {
			mentions: &model.UserMentionMap{
				"non_connected_user": "non_connected_user",
			},
			userSearch:    "non_connected_user",
			expectedError: "the mentioned user is not connected to Jira",
			SetupAPI: func(api *plugintest.API) {
				api.On("LogWarn", mockAnythingOfTypeBatch("string", 5)...)
			},
		},
		"Connected users are shown and returned as Jira Users, when mentioned": {
			mentions: &model.UserMentionMap{
				"connected_user": string(testUser.MattermostUserID)},
			userSearch:     "connected_user",
			expectedResult: &jira.User{AccountID: "test-AccountID"},
			SetupAPI:       func(api *plugintest.API) {},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			defer api.AssertExpectations(t)

			tc.SetupAPI(api)
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)

			user, err := p.GetJiraUserFromMentions(testInstance1.InstanceID, *tc.mentions, tc.userSearch)
			if tc.expectedError != "" {
				assert.Equal(t, tc.expectedError, err.Error())
				assert.Nil(t, user)
				return
			}

			assert.Equal(t, tc.expectedResult, user)
			assert.Nil(t, err)
		})
	}
}
