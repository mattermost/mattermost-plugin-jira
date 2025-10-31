// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/trivago/tgo/tcontainer"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	nonExistantIssueKey   = "FAKE-1"
	noPermissionsIssueKey = "SUDO-1"
	attachCommentErrorKey = "ATTACH-1"
	existingIssueKey      = "REAL-1"
	nonExistantProjectKey = "FP"
	noIssueFoundError     = "We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue."
	noPermissionsError    = "You do not have the appropriate permissions to perform this action. Please contact your Jira administrator."
)

type testClient struct {
	RESTService
	UserService
	ProjectService
	SearchService
	IssueService
}

func (client testClient) GetProject(key string) (*jira.Project, error) {
	if key == nonExistantProjectKey {
		return nil, errors.New("Project " + key + " not found")
	}
	return &jira.Project{
		Key:  key,
		Name: "Test Project",
	}, nil
}

func (client testClient) GetTransitions(issueKey string) ([]jira.Transition, error) {
	if issueKey == nonExistantIssueKey {
		return []jira.Transition{}, errors.New(noIssueFoundError)
	} else if issueKey == noPermissionsIssueKey {
		return []jira.Transition{}, nil
	}

	return []jira.Transition{
		{To: jira.Status{Name: "To Do"}},
		{To: jira.Status{Name: "In Progress"}},
		{To: jira.Status{Name: "In Testing"}},
	}, nil
}

func (client testClient) DoTransition(issueKey string, transitionID string) error {
	return nil
}

func (client testClient) GetIssue(issueKey string, options *jira.GetQueryOptions) (*jira.Issue, error) {
	if issueKey == nonExistantIssueKey {
		return nil, kvstore.ErrNotFound
	}
	return &jira.Issue{
		Key: issueKey,
		Fields: &jira.IssueFields{
			Summary:  "Test Issue Summary",
			Reporter: &jira.User{},
			Status:   &jira.Status{},
		},
	}, nil
}

func (client testClient) AddComment(issueKey string, comment *jira.Comment) (*jira.Comment, error) {
	if issueKey == noPermissionsIssueKey {
		return nil, errors.New("you do not have the permission to comment on this issue")
	} else if issueKey == attachCommentErrorKey {
		return nil, errors.New("unanticipated error")
	}

	return nil, nil
}

func setupTestPlugin(api *plugintest.API) *Plugin {
	api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return()
	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return()

	p := &Plugin{}
	p.SetAPI(api)
	p.initializeRouter()
	p.instanceStore = p.getMockInstanceStoreKV(1)
	p.userStore = getMockUserStoreKV()
	p.client = pluginapi.NewClient(api, p.Driver)

	return p
}

func (client testClient) GetCreateMetaInfo(api plugin.API, options *jira.GetQueryOptions) (*jira.CreateMetaInfo, error) {
	return &jira.CreateMetaInfo{
		Projects: []*jira.MetaProject{
			{
				IssueTypes: []*jira.MetaIssueType{
					{
						Fields: tcontainer.MarshalMap{
							"security": tcontainer.MarshalMap{
								"allowedValues": []interface{}{
									tcontainer.MarshalMap{
										"id": "10001",
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func (client testClient) CreateIssue(issue *jira.Issue) (*jira.Issue, error) {
	// Return a mock created issue
	return &jira.Issue{
		ID:  "10001",
		Key: "TEST-1",
		Fields: &jira.IssueFields{
			Summary: issue.Fields.Summary,
			Project: issue.Fields.Project,
			Type:    issue.Fields.Type,
		},
	}, nil
}

func TestTransitionJiraIssue(t *testing.T) {
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})

	p := setupTestPlugin(api)

	for name, tt := range map[string]struct {
		issueKey    string
		toState     string
		expectedMsg string
		expectedErr error
	}{
		"Transitioning a non existent issue": {
			issueKey:    nonExistantIssueKey,
			toState:     "To Do",
			expectedMsg: "",
			expectedErr: errors.New(noIssueFoundError),
		},
		"Transitioning an issue where user does not have access": {
			issueKey:    noPermissionsIssueKey,
			toState:     "To Do",
			expectedMsg: "",
			expectedErr: errors.New(noPermissionsError),
		},
		"Looking for an invalid state": {
			issueKey:    existingIssueKey,
			toState:     "tofu",
			expectedMsg: "",
			expectedErr: errors.New("\"tofu\" is not a valid state. Please use one of: \"To Do, In Progress, In Testing\""),
		},
		"Matching multiple available states": {
			issueKey:    existingIssueKey,
			toState:     "in",
			expectedMsg: "",
			expectedErr: errors.New("please be more specific, \"in\" matched several states: \"In Progress, In Testing\""),
		},
		"Successfully transitioning to new state": {
			issueKey:    existingIssueKey,
			toState:     "inprog",
			expectedMsg: fmt.Sprintf("[%s](%s/browse/%s) transitioned to `In Progress`", existingIssueKey, mockInstance1URL, existingIssueKey),
			expectedErr: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual, err := p.TransitionIssue(&InTransitionIssue{
				InstanceID:       testInstance1.InstanceID,
				mattermostUserID: "connected_user",
				IssueKey:         tt.issueKey,
				ToState:          tt.toState,
			})
			assert.Equal(t, tt.expectedMsg, actual)
			if tt.expectedErr != nil {
				assert.Error(t, tt.expectedErr, err)
			}
		})
	}
}

func TestRouteIssueTransition(t *testing.T) {
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})
	api.On("LogWarn", "ERROR: ", "Status", "401", "Error", "", "Path", "/api/v2/transition", "Method", "POST", "query", "").Return(nil)
	api.On("LogWarn", "ERROR: ", "Status", "500", "Error", "", "Path", "/api/v2/transition", "Method", "POST", "query", "").Return(nil)
	api.On("LogWarn", "Recovered from a panic", "url", "/api/v2/transition", "error", mock.Anything, "stack", mock.Anything).Return(nil)

	p := setupTestPlugin(api)

	for name, tt := range map[string]struct {
		bb           []byte
		request      *model.PostActionIntegrationRequest
		expectedCode int
	}{
		"No request data": {
			request:      nil,
			expectedCode: http.StatusUnauthorized,
		},
		"No UserID": {
			request: &model.PostActionIntegrationRequest{
				UserId: "",
			},
			expectedCode: http.StatusUnauthorized,
		},
		"No issueKey": {
			request: &model.PostActionIntegrationRequest{
				UserId: "userID",
			},
			expectedCode: http.StatusInternalServerError,
		},
		"No selected_option": {
			request: &model.PostActionIntegrationRequest{
				UserId:  "userID",
				Context: map[string]interface{}{"issueKey": "Some-Key"},
			},
			expectedCode: http.StatusInternalServerError,
		},
	} {
		t.Run(name, func(t *testing.T) {
			bb, err := json.Marshal(tt.request)
			assert.Nil(t, err)

			request := httptest.NewRequest("POST", makeAPIRoute(routeIssueTransition), strings.NewReader(string(bb)))
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tt.expectedCode, w.Result().StatusCode, "no request data")
		})
	}
}

func TestRouteShareIssuePublicly(t *testing.T) {
	validUserID := "1"
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)
	api.On("DeleteEphemeralPost", validUserID, "").Return()
	api.On("LogWarn", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()
	api.On("LogWarn", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()

	p := setupTestPlugin(api)

	for name, tt := range map[string]struct {
		bb           []byte
		request      *model.PostActionIntegrationRequest
		expectedCode int
	}{
		"No request data": {
			request:      nil,
			expectedCode: http.StatusUnauthorized,
		},
		"No UserID": {
			request: &model.PostActionIntegrationRequest{
				UserId: "",
			},
			expectedCode: http.StatusUnauthorized,
		},
		"No issueKey": {
			request: &model.PostActionIntegrationRequest{
				UserId: "userID",
			},
			expectedCode: http.StatusInternalServerError,
		},
		"No instanceId": {
			request: &model.PostActionIntegrationRequest{
				UserId: "userID",
				Context: map[string]interface{}{
					"issue_key": "TEST-10",
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
		"No connection": {
			request: &model.PostActionIntegrationRequest{
				UserId: "userID",
				Context: map[string]interface{}{
					"issue_key":   "TEST-10",
					"instance_id": "id",
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
		"Happy Path": {
			request: &model.PostActionIntegrationRequest{
				UserId: validUserID,
				Context: map[string]interface{}{
					"issue_key":   "TEST-10",
					"instance_id": testInstance1.InstanceID.String(),
				},
			},
			expectedCode: http.StatusOK,
		},
	} {
		t.Run(name, func(t *testing.T) {
			bb, err := json.Marshal(tt.request)
			assert.Nil(t, err)

			request := httptest.NewRequest("POST", makeAPIRoute(routeSharePublicly), strings.NewReader(string(bb)))
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tt.expectedCode, w.Result().StatusCode, "no request data")
		})
	}
}

func TestShouldReceiveNotification(t *testing.T) {
	cs := ConnectionSettings{}
	cs.RolesForDMNotification = make(map[string]bool)
	cs.RolesForDMNotification[assigneeRole] = true
	cs.RolesForDMNotification[mentionRole] = true
	cs.RolesForDMNotification[reporterRole] = false
	cs.RolesForDMNotification[watchingRole] = false
	cs.Notifications = true
	for name, tt := range map[string]struct {
		role         string
		notification bool
	}{
		assigneeRole: {
			role:         assigneeRole,
			notification: true,
		},
		mentionRole: {
			role:         mentionRole,
			notification: true,
		},
		reporterRole: {
			role:         reporterRole,
			notification: false,
		},
		watchingRole: {
			role:         watchingRole,
			notification: false,
		},
		"No Role": {
			role:         "",
			notification: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			val := cs.ShouldReceiveNotification(tt.role)
			assert.Equal(t, tt.notification, val)
		})
	}
}

func TestFetchConnectedUser(t *testing.T) {
	p := setupTestPlugin(&plugintest.API{})

	for name, tt := range map[string]struct {
		instanceID  types.ID
		client      Client
		connection  *Connection
		wh          webhook
		expectedErr error
	}{
		"Success": {
			instanceID: testInstance1.InstanceID,
			client:     testClient{},
			connection: &Connection{
				Settings: &ConnectionSettings{
					Notifications: true,
					RolesForDMNotification: map[string]bool{
						assigneeRole: true,
						mentionRole:  true,
						reporterRole: true,
						watchingRole: true,
					},
				},
				User: jira.User{
					AccountID: "test-AccountID",
				},
			},
			wh: webhook{
				JiraWebhook: &JiraWebhook{
					Issue: jira.Issue{
						Fields: &jira.IssueFields{
							Creator: &jira.User{},
						},
					},
				},
			},
			expectedErr: nil,
		},
		"Issue Field not found": {
			instanceID: testInstance1.InstanceID,
			client:     nil,
			connection: nil,
			wh: webhook{
				JiraWebhook: &JiraWebhook{
					Issue: jira.Issue{},
				},
			},
			expectedErr: nil,
		},
		"Unable to load instance": {
			instanceID: "test-instanceID",
			client:     nil,
			connection: nil,
			wh: webhook{
				JiraWebhook: &JiraWebhook{
					Issue: jira.Issue{
						Fields: &jira.IssueFields{
							Creator: &jira.User{},
						},
					},
				},
			},
			expectedErr: errors.New(fmt.Sprintf("instance %q not found", "test-instanceID")),
		},
	} {
		t.Run(name, func(t *testing.T) {
			client, connection, error := tt.wh.fetchConnectedUser(p, tt.instanceID)
			assert.Equal(t, tt.connection, connection)
			assert.Equal(t, tt.client, client)
			if tt.expectedErr != nil {
				assert.Error(t, tt.expectedErr, error)
			}
		})
	}
}

func TestApplyReporterNotification(t *testing.T) {
	p := setupTestPlugin(&plugintest.API{})

	wh := &webhook{
		eventTypes: map[string]bool{createdCommentEvent: true},
		JiraWebhook: &JiraWebhook{
			Comment: jira.Comment{
				UpdateAuthor: jira.User{},
			},
			Issue: jira.Issue{
				Key: "test-key",
				Fields: &jira.IssueFields{
					Type: jira.IssueType{
						Name: "Story",
					},
					Summary: "",
				},
				Self: "test-self",
			},
		},
	}
	for name, tt := range map[string]struct {
		instanceID         types.ID
		reporter           *jira.User
		totalNotifications int
	}{
		"Success": {
			instanceID:         testInstance1.InstanceID,
			reporter:           &jira.User{},
			totalNotifications: 1,
		},
		"Unable to load instance": {
			instanceID: "test-instanceID",
			reporter:   &jira.User{},
		},
		"Reporter is nil": {
			instanceID: testInstance1.InstanceID,
			reporter:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			wh.notifications = []webhookUserNotification{}
			p.applyReporterNotification(wh, tt.instanceID, tt.reporter)
			assert.Equal(t, len(wh.notifications), tt.totalNotifications)
		})
	}
}

func TestGetUserSetting(t *testing.T) {
	p := setupTestPlugin(&plugintest.API{})

	jiraAccountID := "test-jiraAccountID"
	jiraUsername := "test-jiraUsername"

	for name, tt := range map[string]struct {
		wh          *webhook
		instanceID  types.ID
		connection  *Connection
		expectedErr error
	}{
		"Success": {
			wh:         &webhook{},
			instanceID: testInstance1.InstanceID,
			connection: &Connection{
				User: jira.User{AccountID: "test-AccountID"},
				Settings: &ConnectionSettings{
					Notifications: true,
					RolesForDMNotification: (map[string]bool{
						assigneeRole: true,
						mentionRole:  true,
						reporterRole: true,
						watchingRole: true,
					}),
				},
			},
			expectedErr: nil,
		},
		"Unable to load instance": {
			wh:          &webhook{},
			instanceID:  "instanceID",
			connection:  nil,
			expectedErr: errors.New("instance " + fmt.Sprintf("\"%s\"", "instanceID") + " not found"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			connection, error := p.GetUserSetting(tt.wh, tt.instanceID, jiraAccountID, jiraUsername)
			assert.Equal(t, tt.connection, connection)
			if tt.expectedErr != nil {
				assert.Error(t, tt.expectedErr, error)
			}
		})
	}
}

func TestRouteAttachCommentToIssue(t *testing.T) {
	api := &plugintest.API{}
	api.On("GetPost", "error_post").Return(nil, &model.AppError{Id: "1"})
	api.On("GetPost", "post_not_found").Return(nil, (*model.AppError)(nil))
	api.On("GetPost", "valid_post").Return(&model.Post{
		UserId: "userID",
	}, nil)
	api.On("GetPost", "0").Return(&model.Post{
		UserId: "user_not_found",
	}, nil)
	api.On("GetUser", "userID").Return(&model.User{}, nil)
	// Ensure GetUser for "user_not_found" returns an error or nil
	api.On("GetUser", "user_not_found").Return(nil, &model.AppError{Id: "2"})
	api.On("LogWarn", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})

	p := setupTestPlugin(api)
	p.updateConfig(func(conf *config) {
		conf.mattermostSiteURL = "https://somelink.com"
	})

	type requestStruct struct {
		PostID      string `json:"post_id"`
		InstanceID  string `json:"instance_id"`
		CurrentTeam string `json:"current_team"`
		IssueKey    string `json:"issueKey"`
	}

	for name, tt := range map[string]struct {
		method       string
		header       string
		request      *requestStruct
		expectedCode int
	}{
		"Wrong method": {
			method:       "GET",
			header:       "",
			request:      &requestStruct{},
			expectedCode: http.StatusNotFound,
		},
		"No header": {
			method:       "POST",
			header:       "",
			request:      &requestStruct{},
			expectedCode: http.StatusUnauthorized,
		},
		"User not found": {
			method:       "POST",
			header:       "nobody",
			request:      &requestStruct{},
			expectedCode: http.StatusInternalServerError,
		},
		"Failed to load post": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID: "error_post",
			},
			expectedCode: http.StatusInternalServerError,
		},
		"Post not found": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID: "post_not_found",
			},
			expectedCode: http.StatusInternalServerError,
		},
		"Post user not found": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID: "0",
			},
			expectedCode: http.StatusInternalServerError,
		},
		"No permissions to comment on issue": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID:   "valid_post",
				IssueKey: noPermissionsIssueKey,
			},
			expectedCode: http.StatusInternalServerError,
		},
		"Failed to attach the comment": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID:   "valid_post",
				IssueKey: attachCommentErrorKey,
			},
			expectedCode: http.StatusInternalServerError,
		},
		"Successfully created notification post": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID:   "valid_post",
				IssueKey: existingIssueKey,
			},
			expectedCode: http.StatusOK,
		},
	} {
		t.Run(name, func(t *testing.T) {
			p.initializeRouter()

			tt.request.InstanceID = testInstance1.InstanceID.String()
			bb, err := json.Marshal(tt.request)
			assert.Nil(t, err, name)

			request := httptest.NewRequest(tt.method, makeAPIRoute(routeAPIAttachCommentToIssue), strings.NewReader(string(bb)))
			request.Header.Add("Mattermost-User-Id", tt.header)
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tt.expectedCode, w.Result().StatusCode, name)
		})
	}
}

func TestCreateIssue(t *testing.T) {
	api := &plugintest.API{}

	// Mock post that exists and user has access to
	api.On("GetPost", "accessible_post_id").Return(&model.Post{
		Id:        "accessible_post_id",
		UserId:    "connected_user",
		ChannelId: "channel_id_1",
		Message:   "Test message",
	}, (*model.AppError)(nil))

	// Mock post that exists but user doesn't have access to
	api.On("GetPost", "inaccessible_post_id").Return(&model.Post{
		Id:        "inaccessible_post_id",
		UserId:    "other_user",
		ChannelId: "private_channel_id",
		Message:   "Private message",
	}, (*model.AppError)(nil))

	// Mock reply post (threaded message) that user has access to
	api.On("GetPost", "accessible_reply_post_id").Return(&model.Post{
		Id:        "accessible_reply_post_id",
		UserId:    "connected_user",
		ChannelId: "channel_id_1",
		RootId:    "root_post_id",
		Message:   "Reply message",
	}, (*model.AppError)(nil))

	// Mock reply post in private channel
	api.On("GetPost", "inaccessible_reply_post_id").Return(&model.Post{
		Id:        "inaccessible_reply_post_id",
		UserId:    "other_user",
		ChannelId: "private_channel_id",
		RootId:    "private_root_post_id",
		Message:   "Private reply",
	}, (*model.AppError)(nil))

	// Mock post that doesn't exist
	api.On("GetPost", "nonexistent_post_id").Return(nil, &model.AppError{
		Id:      "app.post.get.app_error",
		Message: "Post not found",
	})

	// Mock DM channel post
	api.On("GetPost", "dm_post_id").Return(&model.Post{
		Id:        "dm_post_id",
		UserId:    "other_user",
		ChannelId: "dm_channel_id",
		Message:   "DM message",
	}, (*model.AppError)(nil))

	// Mock GetMember: user IS a member of channel_id_1
	api.On("GetChannelMember", "channel_id_1", "connected_user").Return(&model.ChannelMember{
		ChannelId: "channel_id_1",
		UserId:    "connected_user",
	}, (*model.AppError)(nil))

	// Mock GetMember: user is NOT a member of private_channel_id
	api.On("GetChannelMember", "private_channel_id", "connected_user").Return(nil, &model.AppError{
		Id:      "api.context.permissions.app_error",
		Message: "User does not have access to this channel",
	})

	// Mock GetMember: user is NOT a member of dm_channel_id
	api.On("GetChannelMember", "dm_channel_id", "connected_user").Return(nil, &model.AppError{
		Id:      "api.context.permissions.app_error",
		Message: "User does not have access to this channel",
	})

	// Mock successful issue creation
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, (*model.AppError)(nil))
	api.On("PublishWebSocketEvent", "update_defaults", mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("*model.WebsocketBroadcast"))

	tests := map[string]struct {
		postID         string
		channelID      string
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		"Create issue without post - should succeed": {
			postID:         "",
			channelID:      "channel_id_1",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		"Create issue with accessible post - should succeed": {
			postID:         "accessible_post_id",
			channelID:      "channel_id_1",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		"Create issue with inaccessible post - should fail with 403": {
			postID:         "inaccessible_post_id",
			channelID:      "channel_id_1",
			expectedStatus: http.StatusForbidden,
			expectError:    true,
			errorContains:  "User does not have access to this post",
		},
		"SECURITY: Bypass attempt - inaccessible post with accessible channelID in request should fail": {
			postID:         "inaccessible_post_id", // post is in private_channel_id
			channelID:      "channel_id_1",         // attacker provides accessible channel ID
			expectedStatus: http.StatusForbidden,
			expectError:    true,
			errorContains:  "User does not have access to this post",
		},
		"Create issue with accessible reply post - should succeed": {
			postID:         "accessible_reply_post_id",
			channelID:      "channel_id_1",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		"Create issue with inaccessible reply post - should fail with 403": {
			postID:         "inaccessible_reply_post_id",
			channelID:      "channel_id_1",
			expectedStatus: http.StatusForbidden,
			expectError:    true,
			errorContains:  "User does not have access to this post",
		},
		"Create issue with nonexistent post - should fail with 500": {
			postID:         "nonexistent_post_id",
			channelID:      "channel_id_1",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
			errorContains:  "failed to load post",
		},
		"Create issue with DM post user doesn't have access to - should fail with 403": {
			postID:         "dm_post_id",
			channelID:      "channel_id_1",
			expectedStatus: http.StatusForbidden,
			expectError:    true,
			errorContains:  "User does not have access to this post",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := Plugin{}
			p.initializeRouter()
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)
			p.updateConfig(func(conf *config) {
				conf.mattermostSiteURL = "https://somelink.com"
			})
			p.userStore = getMockUserStoreKV()
			p.instanceStore = p.getMockInstanceStoreKV(1)

			// Create the InCreateIssue input
			in := &InCreateIssue{
				PostID:           tt.postID,
				CurrentTeam:      "test_team",
				ChannelID:        tt.channelID,
				mattermostUserID: "connected_user",
				InstanceID:       testInstance1.InstanceID,
				Fields: jira.IssueFields{
					Project: jira.Project{
						Key: mockProjectKey,
					},
					Type: jira.IssueType{
						ID: "10001",
					},
					Summary:     "Test Issue",
					Description: "Test description",
				},
			}

			// Call CreateIssue
			issue, statusCode, err := p.CreateIssue(in)

			// Assertions
			assert.Equal(t, tt.expectedStatus, statusCode, "Expected status code to match")

			if tt.expectError {
				assert.Error(t, err, "Expected an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected text")
				}
				assert.Nil(t, issue, "Issue should be nil when error occurs")
			} else {
				assert.NoError(t, err, "Expected no error")
				assert.NotNil(t, issue, "Issue should not be nil on success")
			}
		})
	}
}

func TestRouteCreateIssue(t *testing.T) {
	api := &plugintest.API{}

	api.On("LogWarn", mockAnythingOfTypeBatch("string", 13)...).Return(nil)
	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)

	// Mock post that exists and user has access to
	api.On("GetPost", "accessible_post_id").Return(&model.Post{
		Id:        "accessible_post_id",
		UserId:    "connected_user",
		ChannelId: "channel_id_1",
		Message:   "Test message",
	}, (*model.AppError)(nil))

	// Mock post that exists but user doesn't have access to
	api.On("GetPost", "inaccessible_post_id").Return(&model.Post{
		Id:        "inaccessible_post_id",
		UserId:    "other_user",
		ChannelId: "private_channel_id",
		Message:   "Private message",
	}, (*model.AppError)(nil))

	// Mock GetMember: user IS a member of channel_id_1
	api.On("GetChannelMember", "channel_id_1", "connected_user").Return(&model.ChannelMember{
		ChannelId: "channel_id_1",
		UserId:    "connected_user",
	}, (*model.AppError)(nil))

	// Mock GetMember: user is NOT a member of private_channel_id
	api.On("GetChannelMember", "private_channel_id", "connected_user").Return(nil, &model.AppError{
		Id:      "api.context.permissions.app_error",
		Message: "User does not have access to this channel",
	})

	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, (*model.AppError)(nil))
	api.On("PublishWebSocketEvent", "update_defaults", mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("*model.WebsocketBroadcast"))

	type requestStruct struct {
		PostID      string           `json:"post_id"`
		InstanceID  string           `json:"instance_id"`
		CurrentTeam string           `json:"current_team"`
		ChannelID   string           `json:"channel_id"`
		Fields      jira.IssueFields `json:"fields"`
	}

	tests := map[string]struct {
		method       string
		userID       string
		request      *requestStruct
		expectedCode int
	}{
		"No user header": {
			method:       "POST",
			userID:       "",
			request:      &requestStruct{},
			expectedCode: http.StatusUnauthorized,
		},
		"Create issue without post - should succeed": {
			method: "POST",
			userID: "connected_user",
			request: &requestStruct{
				PostID:      "",
				CurrentTeam: "test_team",
				ChannelID:   "channel_id_1",
				Fields: jira.IssueFields{
					Project: jira.Project{
						Key: mockProjectKey,
					},
					Type: jira.IssueType{
						ID: "10001",
					},
					Summary:     "Test Issue",
					Description: "Test description",
				},
			},
			expectedCode: http.StatusOK,
		},
		"Create issue with accessible post - should succeed": {
			method: "POST",
			userID: "connected_user",
			request: &requestStruct{
				PostID:      "accessible_post_id",
				CurrentTeam: "test_team",
				ChannelID:   "channel_id_1",
				Fields: jira.IssueFields{
					Project: jira.Project{
						Key: mockProjectKey,
					},
					Type: jira.IssueType{
						ID: "10001",
					},
					Summary:     "Test Issue",
					Description: "Test description",
				},
			},
			expectedCode: http.StatusOK,
		},
		"Create issue with inaccessible post - should fail with 403": {
			method: "POST",
			userID: "connected_user",
			request: &requestStruct{
				PostID:      "inaccessible_post_id",
				CurrentTeam: "test_team",
				ChannelID:   "channel_id_1",
				Fields: jira.IssueFields{
					Project: jira.Project{
						Key: mockProjectKey,
					},
					Type: jira.IssueType{
						ID: "10001",
					},
					Summary:     "Test Issue",
					Description: "Test description",
				},
			},
			expectedCode: http.StatusForbidden,
		},
		"SECURITY: Bypass attempt via HTTP route - should fail with 403": {
			method: "POST",
			userID: "connected_user",
			request: &requestStruct{
				PostID:      "inaccessible_post_id", // post is in private_channel_id
				CurrentTeam: "test_team",
				ChannelID:   "channel_id_1", // attacker provides accessible channel ID
				Fields: jira.IssueFields{
					Project: jira.Project{
						Key: mockProjectKey,
					},
					Type: jira.IssueType{
						ID: "10001",
					},
					Summary:     "Test Issue",
					Description: "Test description",
				},
			},
			expectedCode: http.StatusForbidden,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := Plugin{}
			p.initializeRouter()
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)
			p.updateConfig(func(conf *config) {
				conf.mattermostSiteURL = "https://somelink.com"
			})
			p.userStore = getMockUserStoreKV()
			p.instanceStore = p.getMockInstanceStoreKV(1)

			tt.request.InstanceID = testInstance1.InstanceID.String()
			bb, err := json.Marshal(tt.request)
			assert.Nil(t, err, name)

			request := httptest.NewRequest(tt.method, makeAPIRoute(routeAPICreateIssue), strings.NewReader(string(bb)))
			request.Header.Add("Mattermost-User-Id", tt.userID)
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tt.expectedCode, w.Result().StatusCode, name)
		})
	}
}
