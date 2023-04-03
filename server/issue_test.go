package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
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

func TestTransitionJiraIssue(t *testing.T) {
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})
	p := Plugin{}
	p.initializeRouter()
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.userStore = getMockUserStoreKV()
	p.instanceStore = p.getMockInstanceStoreKV(1)

	tests := map[string]struct {
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
	}

	for name, tt := range tests {
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

	api.On("LogWarn", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)

	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})

	p := Plugin{}
	p.initializeRouter()
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.userStore = getMockUserStoreKV()

	tests := map[string]struct {
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
	}
	for name, tt := range tests {
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
	p := Plugin{}
	p.initializeRouter()
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(&model.Post{})
	api.On("LogWarn", mockAnythingOfTypeBatch("string", 13)...).Return(nil)
	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)
	api.On("DeleteEphemeralPost", validUserID, "").Return()
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.instanceStore = p.getMockInstanceStoreKV(1)
	p.userStore = getMockUserStoreKV()

	tests := map[string]struct {
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
	}
	for name, tt := range tests {
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

func TestRouteAttachCommentToIssue(t *testing.T) {
	api := &plugintest.API{}

	api.On("LogWarn", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)

	api.On("GetPost", "error_post").Return(nil, &model.AppError{Id: "1"})
	api.On("GetPost", "post_not_found").Return(nil, (*model.AppError)(nil))

	api.On("GetPost", "0").Return(&model.Post{UserId: "0"}, (*model.AppError)(nil))
	api.On("GetUser", "0").Return(nil, &model.AppError{Id: "1"})

	api.On("GetPost", "1").Return(&model.Post{UserId: "1"}, (*model.AppError)(nil))
	api.On("GetUser", "1").Return(&model.User{Username: "username"}, (*model.AppError)(nil))
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, (*model.AppError)(nil))

	api.On("PublishWebSocketEvent", "update_defaults", mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("*model.WebsocketBroadcast"))

	type requestStruct struct {
		PostID      string `json:"post_id"`
		InstanceID  string `json:"instance_id"`
		CurrentTeam string `json:"current_team"`
		IssueKey    string `json:"issueKey"`
	}

	tests := map[string]struct {
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
			assert.Nil(t, err)

			request := httptest.NewRequest(tt.method, makeAPIRoute(routeAPIAttachCommentToIssue), strings.NewReader(string(bb)))
			request.Header.Add("Mattermost-User-Id", tt.header)
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tt.expectedCode, w.Result().StatusCode, "no request data")
		})
	}
}
