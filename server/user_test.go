package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-server/v6/model"
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
			settings:       ConnectionSettings{Notifications: false},
			expectedOutput: "\tNotifications: off",
		},
		"notifications off": {
			settings:       ConnectionSettings{Notifications: true},
			expectedOutput: "\tNotifications: on",
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

	api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)

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

func TestGetJiraUserFromMentions(t *testing.T) {
	p := Plugin{}
	p.userStore = getMockUserStoreKV()
	p.instanceStore = p.getMockInstanceStoreKV(1)
	testUser, _ := p.userStore.LoadUser("connected_user")

	tests := map[string]struct {
		mentions       *model.UserMentionMap
		userSearch     string
		expectedResult *jira.User
		expectedError  string
	}{
		"if no mentions, no users are returned": {
			mentions:       &model.UserMentionMap{},
			userSearch:     "join",
			expectedResult: nil,
			expectedError: "The user mentioned was not found",
		},
		"non connected user won't appear when mentioned": {
			mentions: &model.UserMentionMap{
				"non_connected_user": "non_connected_user",
			},
			userSearch:     "non_connected_user",
			expectedResult: nil,
			expectedError: "The user mentioned is not connected to Jira",
		},
		"Connected users are shown and returned as JiraUsers when mentioned": {
			mentions: &model.UserMentionMap{
				"connected_user": string(testUser.MattermostUserID)},
			userSearch:     "connected_user",
			expectedResult: &jira.User{AccountID: "test"},
			expectedError: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			user, err := p.GetJiraUserFromMentions(
				testInstance1.InstanceID, *tc.mentions, tc.userSearch)

			if err != nil {
				assert.Equal(t, tc.expectedError, err.Error())
			}

			assert.Equal(t, tc.expectedResult, user)
		})
	}
}
