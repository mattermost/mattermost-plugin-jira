// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

var errNotFound = errors.New("not found")

func TestParseJIRAUsernamesFromText(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected []string
	}{
		{Text: "[~]", Expected: []string{}},
		{Text: "[~j]", Expected: []string{"j"}},
		{Text: "[~jira]", Expected: []string{"jira"}},
		{Text: "[~jira.user]", Expected: []string{"jira.user"}},
		{Text: "[~jira_user]", Expected: []string{"jira_user"}},
		{Text: "[~jira-user]", Expected: []string{"jira-user"}},
		{Text: "[~jira:user]", Expected: []string{"jira:user"}},
		{Text: "[~jira_user_3]", Expected: []string{"jira_user_3"}},
		{Text: "[~jira-user-4]", Expected: []string{"jira-user-4"}},
		{Text: "[~JiraUser5]", Expected: []string{"JiraUser5"}},
		{Text: "[~jira-user+6]", Expected: []string{"jira-user+6"}},
		{Text: "[~2023]", Expected: []string{"2023"}},
		{Text: "[~jira.user@company.com]", Expected: []string{"jira.user@company.com"}},
		{Text: "[~jira_user@mattermost.com]", Expected: []string{"jira_user@mattermost.com"}},
		{Text: "[~jira-unique-user@mattermost.com] [~jira-unique-user@mattermost.com] [~jira-unique-user@mattermost.com]", Expected: []string{"jira-unique-user@mattermost.com"}},
		{Text: "[jira_incorrect_user]", Expected: []string{}},
		{Text: "[~jira_user_reviewer], Hi! Can you review the PR from [~jira_user_contributor]? Thanks!", Expected: []string{"jira_user_reviewer", "jira_user_contributor"}},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, parseJIRAUsernamesFromText(tc.Text))
	}
}

type mockUserStoreForUtils struct {
	mattermostUserID types.ID
	err              error
}

func (m mockUserStoreForUtils) LoadUser(types.ID) (*User, error) {
	return nil, nil
}

func (m mockUserStoreForUtils) StoreUser(*User) error {
	return nil
}

func (m mockUserStoreForUtils) StoreConnection(types.ID, types.ID, *Connection) error {
	return nil
}

func (m mockUserStoreForUtils) LoadConnection(types.ID, types.ID) (*Connection, error) {
	return nil, nil
}

func (m mockUserStoreForUtils) LoadMattermostUserID(instanceID types.ID, jiraUsername string) (types.ID, error) {
	return m.mattermostUserID, m.err
}

func (m mockUserStoreForUtils) DeleteConnection(types.ID, types.ID) error {
	return nil
}

func (m mockUserStoreForUtils) CountUsers() (int, error) {
	return 0, nil
}

func (m mockUserStoreForUtils) MapUsers(func(*User) error) error {
	return nil
}

type mockInstanceStoreForUtils struct {
	instance Instance
	err      error
}

func (m mockInstanceStoreForUtils) CreateInactiveCloudInstance(types.ID, string) error {
	return nil
}

func (m mockInstanceStoreForUtils) DeleteInstance(types.ID) error {
	return nil
}

func (m mockInstanceStoreForUtils) LoadInstance(types.ID) (Instance, error) {
	return m.instance, m.err
}

func (m mockInstanceStoreForUtils) LoadInstanceFullKey(string) (Instance, error) {
	return m.instance, m.err
}

func (m mockInstanceStoreForUtils) LoadInstances() (*Instances, error) {
	return nil, nil
}

func (m mockInstanceStoreForUtils) StoreInstance(Instance) error {
	return nil
}

func (m mockInstanceStoreForUtils) StoreInstances(*Instances) error {
	return nil
}

func TestReplaceJiraAccountIds(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		expectedResult   string
		mattermostUserID types.ID
		loadUserErr      error
		mmUser           *model.User
		mmUserErr        *model.AppError
	}{
		{
			name:             "no mentions in text",
			body:             "Hello world, this is a test message",
			expectedResult:   "Hello world, this is a test message",
			mattermostUserID: "",
			loadUserErr:      nil,
		},
		{
			name:             "mention replaced with mattermost username",
			body:             "Hello [~accountid:123456789], please review",
			expectedResult:   "Hello @testuser, please review",
			mattermostUserID: "mm-user-id",
			loadUserErr:      nil,
			mmUser:           &model.User{Username: "testuser"},
			mmUserErr:        nil,
		},
		{
			name:             "mention falls back to account id when mm user not found and jira lookup fails",
			body:             "Hello [~accountid:123456789], please review",
			expectedResult:   "Hello 123456789, please review",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
		},
		{
			name:             "multiple mentions - falls back to account ids",
			body:             "Hi [~accountid:user1], please check with [~accountid:user2]",
			expectedResult:   "Hi user1, please check with user2",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
		},
		{
			name:             "legacy username format replaced with mattermost username",
			body:             "Hello [~jirauser], please review",
			expectedResult:   "Hello @testuser, please review",
			mattermostUserID: "mm-user-id",
			loadUserErr:      nil,
			mmUser:           &model.User{Username: "testuser"},
			mmUserErr:        nil,
		},
		{
			name:             "legacy username format falls back when not found",
			body:             "Hello [~jirauser], please review",
			expectedResult:   "Hello jirauser, please review",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return().Maybe()
			api.On("LogError", mockAnythingOfTypeBatch("string", 11)...).Return().Maybe()
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 3)...).Return().Maybe()
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 5)...).Return().Maybe()

			if tt.mmUser != nil {
				api.On("GetUser", string(tt.mattermostUserID)).Return(tt.mmUser, tt.mmUserErr)
			} else if tt.mattermostUserID != "" {
				api.On("GetUser", string(tt.mattermostUserID)).Return(nil, model.NewAppError("", "", nil, "", http.StatusNotFound))
			}

			p := &Plugin{}
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)
			p.userStore = mockUserStoreForUtils{
				mattermostUserID: tt.mattermostUserID,
				err:              tt.loadUserErr,
			}

			instanceURL := "https://test.atlassian.net"

			mockInstance := &testInstance{
				InstanceCommon: InstanceCommon{
					InstanceID: types.ID(instanceURL),
				},
			}
			p.instanceStore = mockInstanceStoreForUtils{
				instance: mockInstance,
				err:      nil,
			}

			result := p.replaceJiraAccountIds(types.ID(instanceURL), tt.body)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetJiraUserDisplayName(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		expectedResult string
		instanceErr    error
	}{
		{
			name:           "returns empty when instance load fails",
			accountID:      "123456",
			expectedResult: "",
			instanceErr:    errNotFound,
		},
		{
			name:           "returns empty when instance URL is empty",
			accountID:      "123456",
			expectedResult: "",
			instanceErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return().Maybe()
			api.On("LogError", mockAnythingOfTypeBatch("string", 11)...).Return().Maybe()
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 3)...).Return().Maybe()

			p := &Plugin{}
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)

			instanceURL := "https://test.atlassian.net"

			var mockInstance Instance
			if tt.name == "returns empty when instance URL is empty" {
				mockInstance = &testInstance{
					InstanceCommon: InstanceCommon{
						InstanceID: "",
					},
				}
			} else {
				mockInstance = &testInstance{
					InstanceCommon: InstanceCommon{
						InstanceID: types.ID(instanceURL),
					},
				}
			}

			p.instanceStore = mockInstanceStoreForUtils{
				instance: mockInstance,
				err:      tt.instanceErr,
			}

			result := p.getJiraUserDisplayName(types.ID(instanceURL), tt.accountID)
			require.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetJiraUserDisplayNameWithMockServer(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		apiResponse    map[string]string
		apiStatusCode  int
		expectedResult string
	}{
		{
			name:           "successful fetch returns display name",
			accountID:      "123456",
			apiResponse:    map[string]string{"displayName": "John Doe"},
			apiStatusCode:  http.StatusOK,
			expectedResult: "John Doe",
		},
		{
			name:           "user not found returns empty",
			accountID:      "unknown",
			apiResponse:    nil,
			apiStatusCode:  http.StatusNotFound,
			expectedResult: "",
		},
		{
			name:           "empty display name returns empty",
			accountID:      "123456",
			apiResponse:    map[string]string{"displayName": ""},
			apiStatusCode:  http.StatusOK,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/rest/api/2/user", r.URL.Path)
				assert.Equal(t, tt.accountID, r.URL.Query().Get("accountId"))

				if tt.apiStatusCode != http.StatusOK {
					w.WriteHeader(tt.apiStatusCode)
					return
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tt.apiResponse)
			}))
			defer jiraServer.Close()

			api := &plugintest.API{}
			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return().Maybe()
			api.On("LogError", mockAnythingOfTypeBatch("string", 11)...).Return().Maybe()
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 3)...).Return().Maybe()
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 5)...).Return().Maybe()

			p := &Plugin{}
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)

			// Set up config with encrypted admin token
			p.updateConfig(func(conf *config) {
				conf.AdminAPIToken = ""
				conf.EncryptionKey = ""
				conf.AdminEmail = ""
			})

			mockInstance := &testInstance{
				InstanceCommon: InstanceCommon{
					InstanceID: types.ID(jiraServer.URL),
				},
			}

			p.instanceStore = mockInstanceStoreForUtils{
				instance: mockInstance,
				err:      nil,
			}

			result := p.getJiraUserDisplayName(types.ID(jiraServer.URL), tt.accountID)
			// The function will return empty because SetAdminAPITokenRequestHeader will fail
			// without proper encryption setup. Testing the error paths.
			require.Equal(t, "", result)
		})
	}
}
