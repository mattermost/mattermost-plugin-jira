// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"
	"io"
	"net/http"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mattermost/mattermost-plugin-jira/server/telemetry"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/stretchr/testify/require"
)

var errNotFound = errors.New("not found")

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

type mockJiraClient struct {
	mock.Mock
}

func (m *mockJiraClient) RESTGet(endpoint string, params map[string]string, dest interface{}) error {
	args := m.Called(endpoint, params)
	if args.Error(0) == nil {
		if user, ok := dest.(*jira.User); ok {
			*user = args.Get(1).(jira.User)
		}
	}
	return args.Error(0)
}

func (m *mockJiraClient) RESTPostAttachment(issueID string, data io.Reader, name string) (*jira.Attachment, error) {
	return nil, nil
}

func (m *mockJiraClient) GetSelf() (*jira.User, error)                           { return nil, nil }
func (m *mockJiraClient) GetUserGroups(_ *Connection) ([]*jira.UserGroup, error) { return nil, nil }
func (m *mockJiraClient) GetIssue(_ string, _ *jira.GetQueryOptions) (*jira.Issue, error) {
	return nil, nil
}
func (m *mockJiraClient) CreateIssue(_ *jira.Issue) (*jira.Issue, error) { return nil, nil }
func (m *mockJiraClient) AddAttachment(_ pluginapi.Client, _, _ string, _ types.ByteSize) (string, string, string, error) {
	return "", "", "", nil
}
func (m *mockJiraClient) AddComment(_ string, _ *jira.Comment) (*jira.Comment, error) {
	return nil, nil
}
func (m *mockJiraClient) DoTransition(_, _ string) error { return nil }
func (m *mockJiraClient) GetCreateMetaInfo(_ plugin.API, _ *jira.GetQueryOptions) (*jira.CreateMetaInfo, error) {
	return nil, nil
}
func (m *mockJiraClient) GetTransitions(_ string) ([]jira.Transition, error) { return nil, nil }
func (m *mockJiraClient) UpdateAssignee(_ string, _ *jira.User) error        { return nil }
func (m *mockJiraClient) UpdateComment(_ string, _ *jira.Comment) (*jira.Comment, error) {
	return nil, nil
}
func (m *mockJiraClient) SearchIssues(_ string, _ *jira.SearchOptions) ([]jira.Issue, error) {
	return nil, nil
}
func (m *mockJiraClient) SearchUsersAssignableToIssue(_, _ string, _ int) ([]jira.User, error) {
	return nil, nil
}
func (m *mockJiraClient) SearchUsersAssignableInProject(_, _ string, _ int) ([]jira.User, error) {
	return nil, nil
}
func (m *mockJiraClient) SearchAutoCompleteFields(_ map[string]string) (*AutoCompleteResult, error) {
	return nil, nil
}
func (m *mockJiraClient) GetWatchers(_, _ string, _ *Connection) (*jira.Watches, error) {
	return nil, nil
}
func (m *mockJiraClient) GetUserVisibilityGroups(_ map[string]string) (*CommentVisibilityResult, error) {
	return nil, nil
}
func (m *mockJiraClient) GetProject(_ string) (*jira.Project, error) { return nil, nil }
func (m *mockJiraClient) ListProjects(_ string, _ int, _ bool) (jira.ProjectList, error) {
	return nil, nil
}
func (m *mockJiraClient) GetAllProjectKeys() ([]string, error)             { return nil, nil }
func (m *mockJiraClient) GetIssueTypes(_ string) ([]jira.IssueType, error) { return nil, nil }
func (m *mockJiraClient) ListProjectStatuses(_ string) ([]*IssueTypeWithStatuses, error) {
	return nil, nil
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
		instanceType     InstanceType
		jiraClient       func() *mockJiraClient
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
			instanceType:     CloudOAuthInstanceType,
			mmUser:           &model.User{Username: "testuser"},
			mmUserErr:        nil,
		},
		{
			name:             "cloud - mention falls back to jira display name when mm user not found",
			body:             "Hello [~accountid:123456789], please review",
			expectedResult:   "Hello John Doe, please review",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
			instanceType:     CloudOAuthInstanceType,
			jiraClient: func() *mockJiraClient {
				c := &mockJiraClient{}
				c.On("RESTGet", "2/user", map[string]string{"accountId": "123456789"}).Return(nil, jira.User{DisplayName: "John Doe"})
				return c
			},
		},
		{
			name:             "cloud - mention falls back to raw id when jira lookup also fails",
			body:             "Hello [~accountid:123456789], please review",
			expectedResult:   "Hello 123456789, please review",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
			instanceType:     CloudOAuthInstanceType,
			jiraClient: func() *mockJiraClient {
				c := &mockJiraClient{}
				c.On("RESTGet", "2/user", map[string]string{"accountId": "123456789"}).Return(errors.New("not found"), jira.User{})
				return c
			},
		},
		{
			name:             "mention falls back to raw id when no jira client provided",
			body:             "Hello [~accountid:123456789], please review",
			expectedResult:   "Hello 123456789, please review",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
			instanceType:     CloudOAuthInstanceType,
		},
		{
			name:             "cloud - multiple mentions with jira display name fallback",
			body:             "Hi [~accountid:user1], please check with [~accountid:user2]",
			expectedResult:   "Hi Alice, please check with Bob",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
			instanceType:     CloudOAuthInstanceType,
			jiraClient: func() *mockJiraClient {
				c := &mockJiraClient{}
				c.On("RESTGet", "2/user", map[string]string{"accountId": "user1"}).Return(nil, jira.User{DisplayName: "Alice"})
				c.On("RESTGet", "2/user", map[string]string{"accountId": "user2"}).Return(nil, jira.User{DisplayName: "Bob"})
				return c
			},
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
			name:             "server - legacy username format falls back to jira display name",
			body:             "Hello [~jirauser], please review",
			expectedResult:   "Hello Jane Smith, please review",
			mattermostUserID: "",
			loadUserErr:      errNotFound,
			instanceType:     ServerInstanceType,
			jiraClient: func() *mockJiraClient {
				c := &mockJiraClient{}
				c.On("RESTGet", "2/user", map[string]string{"username": "jirauser"}).Return(nil, jira.User{DisplayName: "Jane Smith"})
				return c
			},
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
					Type:       tt.instanceType,
				},
			}
			p.instanceStore = mockInstanceStoreForUtils{
				instance: mockInstance,
				err:      nil,
			}

			var client Client
			if tt.jiraClient != nil {
				client = tt.jiraClient()
			}

			result := p.replaceJiraAccountIds(types.ID(instanceURL), tt.body, client)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetJiraUserDisplayName(t *testing.T) {
	tests := []struct {
		name           string
		userIdentifier string
		isCloud        bool
		expectedResult string
		setupClient    func() *mockJiraClient
	}{
		{
			name:           "returns empty when jira client is nil",
			userIdentifier: "123456",
			isCloud:        true,
			expectedResult: "",
		},
		{
			name:           "cloud - returns display name via accountId",
			userIdentifier: "123456",
			isCloud:        true,
			expectedResult: "John Doe",
			setupClient: func() *mockJiraClient {
				c := &mockJiraClient{}
				c.On("RESTGet", "2/user", map[string]string{"accountId": "123456"}).Return(nil, jira.User{DisplayName: "John Doe"})
				return c
			},
		},
		{
			name:           "server - returns display name via username",
			userIdentifier: "jdoe",
			isCloud:        false,
			expectedResult: "John Doe",
			setupClient: func() *mockJiraClient {
				c := &mockJiraClient{}
				c.On("RESTGet", "2/user", map[string]string{"username": "jdoe"}).Return(nil, jira.User{DisplayName: "John Doe"})
				return c
			},
		},
		{
			name:           "returns empty when REST call fails",
			userIdentifier: "unknown",
			isCloud:        true,
			expectedResult: "",
			setupClient: func() *mockJiraClient {
				c := &mockJiraClient{}
				c.On("RESTGet", "2/user", map[string]string{"accountId": "unknown"}).Return(errors.New("user not found"), jira.User{})
				return c
			},
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

			var client Client
			if tt.setupClient != nil {
				client = tt.setupClient()
			}

			result := p.getJiraUserDisplayName(client, tt.isCloud, tt.userIdentifier)
			require.Equal(t, tt.expectedResult, result)
		})
	}
}
