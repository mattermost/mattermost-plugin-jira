package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
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
					subCommandAssignee: true,
					subCommandMention:  true,
					subCommandReporter: true,
					subCommandWatching: true,
				},
			},
			expectedOutput: "\t- Notifications for assignee: on \n\t- Notifications for mention: on \n\t- Notifications for reporter: on \n\t- Notifications for watching: on",
		},
		"notifications off": {
			settings: ConnectionSettings{
				Notifications: false,
				RolesForDMNotification: map[string]bool{
					subCommandAssignee: false,
					subCommandMention:  false,
					subCommandReporter: false,
					subCommandWatching: false,
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

func TestRouteUserStart(t *testing.T) {
	tests := map[string]struct {
		userID     string
		statusCode int
	}{
		"user connected to jira will re-direct to docs":  {userID: "connected_user", statusCode: http.StatusSeeOther},
		"user not connected to jira will atempt connect": {userID: "non_connected_user", statusCode: http.StatusFound},
	}
	api := &plugintest.API{}

	api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return()

	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return()

	p := Plugin{}
	p.SetAPI(api)

	p.userStore = getMockUserStoreKV()
	p.instanceStore = p.getMockInstanceStoreKV(1)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest("GET", routeUserStart, nil)
			request.Header.Set("Mattermost-User-Id", tc.userID)
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tc.statusCode, w.Result().StatusCode)
		})
	}
}
