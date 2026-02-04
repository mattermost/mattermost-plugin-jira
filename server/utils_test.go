// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mattermost/mattermost-plugin-jira/server/telemetry"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type mockTelemetryTracker struct{}

func (m *mockTelemetryTracker) TrackEvent(event string, properties map[string]interface{}) error {
	return nil
}

func (m *mockTelemetryTracker) TrackUserEvent(event, userID string, properties map[string]interface{}) error {
	return nil
}

func (m *mockTelemetryTracker) ReloadConfig(config telemetry.TrackerConfig) {
}

type mockInstanceStoreWithLoadInstances struct {
	*mockInstanceStore
}

func (m *mockInstanceStoreWithLoadInstances) LoadInstances() (*Instances, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Instances), args.Error(1)
}

type mockUserStoreForTokenExpiry struct {
	mock.Mock
}

func (m *mockUserStoreForTokenExpiry) StoreUser(user *User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *mockUserStoreForTokenExpiry) LoadUser(id types.ID) (*User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockUserStoreForTokenExpiry) LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error) {
	args := m.Called(instanceID, mattermostUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Connection), args.Error(1)
}

func (m *mockUserStoreForTokenExpiry) DeleteConnection(instanceID, mattermostUserID types.ID) error {
	args := m.Called(instanceID, mattermostUserID)
	return args.Error(0)
}

func (m *mockUserStoreForTokenExpiry) StoreConnection(types.ID, types.ID, *Connection) error {
	return nil
}
func (m *mockUserStoreForTokenExpiry) LoadMattermostUserID(types.ID, string) (types.ID, error) {
	return "", nil
}
func (m *mockUserStoreForTokenExpiry) CountUsers() (int, error)         { return 0, nil }
func (m *mockUserStoreForTokenExpiry) MapUsers(func(*User) error) error { return nil }

func TestDisconnectUserDueToExpiredToken(t *testing.T) {
	testMattermostUserID := types.ID("test-mm-user-id")
	testInstanceID := types.ID("https://test-instance.atlassian.net")
	testChannelID := "test-channel-id"
	testBotUserID := "test-bot-user-id"

	tests := []struct {
		name       string
		setupMocks func(*plugintest.API, *mockUserStoreForTokenExpiry, *mockInstanceStoreWithLoadInstances)
	}{
		{
			name: "Happy path - Disconnect succeeds and DM sent",
			setupMocks: func(api *plugintest.API, userStore *mockUserStoreForTokenExpiry, instanceStore *mockInstanceStoreWithLoadInstances) {
				user := NewUser(testMattermostUserID)
				user.ConnectedInstances = NewInstances()
				user.ConnectedInstances.Set(&InstanceCommon{InstanceID: testInstanceID})

				userStore.On("LoadUser", testMattermostUserID).Return(user, nil).Once()

				mockInstance := &testInstance{
					InstanceCommon: InstanceCommon{
						InstanceID: testInstanceID,
					},
				}
				instanceStore.On("LoadInstance", testInstanceID).Return(mockInstance, nil).Once()

				connection := &Connection{
					MattermostUserID: testMattermostUserID,
				}
				userStore.On("LoadConnection", testInstanceID, testMattermostUserID).Return(connection, nil).Once()
				userStore.On("DeleteConnection", testInstanceID, testMattermostUserID).Return(nil).Once()
				userStore.On("StoreUser", mock.AnythingOfType("*main.User")).Return(nil).Once()

				instances := NewInstances()
				instances.Set(&InstanceCommon{InstanceID: testInstanceID})
				// LoadInstances called twice: once in resolveUserInstanceURL, once in GetUserInfo
				instanceStore.On("LoadInstances").Return(instances, nil).Twice()

				api.On("PublishWebSocketEvent", "disconnect", mock.Anything, mock.MatchedBy(func(b *model.WebsocketBroadcast) bool {
					return b.UserId == testMattermostUserID.String()
				})).Return().Once()

				api.On("GetDirectChannel", testMattermostUserID.String(), testBotUserID).Return(&model.Channel{
					Id: testChannelID,
				}, nil).Once()

				api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
					return post.UserId == testBotUserID &&
						post.ChannelId == testChannelID &&
						post.Message == ":warning: Your Jira connection has expired. Please reconnect your account using `/jira connect https://test-instance.atlassian.net`."
				})).Return(&model.Post{}, nil).Once()
			},
		},
		{
			name: "Disconnect fails but DM with manual instructions sent",
			setupMocks: func(api *plugintest.API, userStore *mockUserStoreForTokenExpiry, instanceStore *mockInstanceStoreWithLoadInstances) {
				userStore.On("LoadUser", testMattermostUserID).Return(nil, kvstore.ErrNotFound).Once()

				api.On("GetDirectChannel", testMattermostUserID.String(), testBotUserID).Return(&model.Channel{
					Id: testChannelID,
				}, nil).Once()

				api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
					return post.UserId == testBotUserID &&
						post.ChannelId == testChannelID &&
						post.Message == ":warning: Your Jira connection has expired. Please manually disconnect and reconnect your account using:\n1. `/jira disconnect https://test-instance.atlassian.net`\n2. `/jira connect https://test-instance.atlassian.net`"
				})).Return(&model.Post{}, nil).Once()

				api.On("LogWarn", "Failed to disconnect user after token expiry",
					"mattermostUserID", testMattermostUserID,
					"instanceID", testInstanceID,
					"error", mock.Anything).Return().Once()
			},
		},
		{
			name: "Disconnect fails and DM also fails - only logging",
			setupMocks: func(api *plugintest.API, userStore *mockUserStoreForTokenExpiry, instanceStore *mockInstanceStoreWithLoadInstances) {
				userStore.On("LoadUser", testMattermostUserID).Return(nil, kvstore.ErrNotFound).Once()

				api.On("GetDirectChannel", testMattermostUserID.String(), testBotUserID).Return(nil, &model.AppError{
					Message: "channel not found",
				}).Once()

				api.On("LogWarn", "Failed to disconnect user after token expiry",
					"mattermostUserID", testMattermostUserID,
					"instanceID", testInstanceID,
					"error", mock.Anything).Return().Once()

				api.On("LogWarn", "Failed to send token expiry notification to user after disconnect failure",
					"mattermostUserID", testMattermostUserID,
					"error", mock.Anything).Return().Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &plugintest.API{}
			userStore := &mockUserStoreForTokenExpiry{}
			instanceStore := &mockInstanceStoreWithLoadInstances{
				mockInstanceStore: &mockInstanceStore{},
			}

			p := &Plugin{
				userStore:     userStore,
				instanceStore: instanceStore,
				tracker:       &mockTelemetryTracker{},
			}
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)
			p.updateConfig(func(conf *config) {
				conf.botUserID = testBotUserID
			})

			tt.setupMocks(api, userStore, instanceStore)
			p.disconnectUserDueToExpiredToken(testMattermostUserID, testInstanceID)

			api.AssertExpectations(t)
			userStore.AssertExpectations(t)
			instanceStore.AssertExpectations(t)
		})
	}
}

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
