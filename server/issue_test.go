package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

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
	return nil, nil
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
		Fields: &jira.IssueFields{
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
	p.instanceStore = p.getMockInstanceStoreKV(1)
	p.userStore = getMockUserStoreKV()
	return p
}

func TestTransitionJiraIssue(t *testing.T) {
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(nil)
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
	api.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(nil)
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

			request := httptest.NewRequest("POST", routeIssueTransition, strings.NewReader(string(bb)))
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tt.expectedCode, w.Result().StatusCode, "no request data")
		})
	}
}

func TestRouteShareIssuePublicly(t *testing.T) {
	validUserID := "1"
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)
	api.On("DeleteEphemeralPost", validUserID, "").Return()
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

			request := httptest.NewRequest("POST", routeSharePublicly, strings.NewReader(string(bb)))
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
		notifications: []webhookUserNotification{},
	}
	for name, tt := range map[string]struct {
		instanceID types.ID
		reporter   *jira.User
	}{
		"Success": {
			instanceID: testInstance1.InstanceID,
			reporter:   &jira.User{},
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
			totalNotifications := len(wh.notifications)
			p.applyReporterNotification(wh, tt.instanceID, tt.reporter)
			if tt.reporter == nil || tt.instanceID == "test-instanceID" {
				assert.Equal(t, len(wh.notifications), totalNotifications)
			} else {
				assert.Equal(t, len(wh.notifications), 1+totalNotifications)
			}
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

	api.On("GetPost", "0").Return(&model.Post{UserId: "0"}, (*model.AppError)(nil))
	api.On("GetUser", "0").Return(nil, &model.AppError{Id: "1"})

	api.On("GetPost", "1").Return(&model.Post{UserId: "1"}, (*model.AppError)(nil))
	api.On("GetUser", "1").Return(&model.User{Username: "username"}, (*model.AppError)(nil))

	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, (*model.AppError)(nil))

	api.On("PublishWebSocketEvent", "update_defaults", mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("*model.WebsocketBroadcast"))

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
			expectedCode: http.StatusMethodNotAllowed,
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
				PostID:   "1",
				IssueKey: noPermissionsIssueKey,
			},
			expectedCode: http.StatusInternalServerError,
		},
		"Failed to attach the comment": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID:   "1",
				IssueKey: attachCommentErrorKey,
			},
			expectedCode: http.StatusInternalServerError,
		},
		"Successfully created notification post": {
			method: "POST",
			header: "1",
			request: &requestStruct{
				PostID:   "1",
				IssueKey: existingIssueKey,
			},
			expectedCode: http.StatusOK,
		},
	} {
		t.Run(name, func(t *testing.T) {
			tt.request.InstanceID = testInstance1.InstanceID.String()
			bb, err := json.Marshal(tt.request)
			assert.Nil(t, err)

			request := httptest.NewRequest(tt.method, routeAPIAttachCommentToIssue, strings.NewReader(string(bb)))
			request.Header.Add("Mattermost-User-Id", tt.header)
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tt.expectedCode, w.Result().StatusCode, "no request data")
		})
	}
}
