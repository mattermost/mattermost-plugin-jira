// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
)

const (
	MockInstanceID      = "mockInstanceID"
	MockAPIToken        = "mockAPIToken"
	MockAdminEmail      = "mockadmin@email.com"
	MockBaseURL         = "mockBaseURL"
	MockASCKey          = "mockAtlassianSecurityContextKey"
	MockASCClientKey    = "mockAtlassianSecurityContextClientKey"
	MockASCSharedSecret = "mockAtlassianSecurityContextSharedSecret" // #nosec G101: Potential hardcoded credentials - This is a mock for testing purposes
)

func validRequestBody() io.ReadCloser {
	if f, err := os.Open("testdata/webhook-issue-created.json"); err != nil {
		panic(err)
	} else {
		return f
	}
}

type TestConfiguration struct {
	Secret   string
	UserName string
}

func TestPlugin(t *testing.T) {
	validConfiguration := TestConfiguration{
		Secret: "thesecret",
	}

	for name, tc := range map[string]struct {
		Configuration      TestConfiguration
		Request            *http.Request
		CreatePostError    *model.AppError
		ExpectedStatusCode int
	}{
		"NoConfiguration": {
			Configuration:      TestConfiguration{},
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusForbidden,
		},
		"NoChannel": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusBadRequest,
		},
		"NoTeam": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusBadRequest,
		},
		"WrongSecret": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=notthesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusForbidden,
		},
		"InvalidBody": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", io.NopCloser(bytes.NewBufferString("foo"))),
			ExpectedStatusCode: http.StatusBadRequest,
		},
		"UnknownJSONPayload": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", io.NopCloser(bytes.NewBufferString("{}"))),
			ExpectedStatusCode: http.StatusBadRequest,
		},
		"InvalidChannel": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=notthechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusBadRequest,
		},
		"InvalidTeam": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=nottheteam&channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusBadRequest,
		},
		"ValidRequest": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusOK,
		},
		"CreatePostError": {
			Configuration:      validConfiguration,
			CreatePostError:    model.NewAppError("foo", "bar", nil, "", http.StatusInternalServerError),
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusInternalServerError,
		},
		"WrongMethod": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("GET", "/webhook", validRequestBody()),
			ExpectedStatusCode: http.StatusMethodNotAllowed,
		},
		"WrongPath": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/not-webhook", validRequestBody()),
			ExpectedStatusCode: http.StatusNotFound,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 10)...).Return(nil)
			api.On("LogWarn", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

			api.On("KVGet", mock.AnythingOfType("string")).Return(make([]byte, 0), (*model.AppError)(nil))
			api.On("GetDirectChannel", mockAnythingOfTypeBatch("string", 2)...).Return(
				&model.Channel{}, (*model.AppError)(nil))
			api.On("GetUserByUsername", "theuser").Return(&model.User{
				Id: "theuserid",
			}, (*model.AppError)(nil))
			api.On("GetUserByUsername", "nottheuser").Return((*model.User)(nil), model.NewAppError("foo", "bar", nil, "", http.StatusBadRequest))

			api.On("GetChannelByNameForTeamName", "nottheteam", "thechannel", false).Return((*model.Channel)(nil), model.NewAppError("foo", "bar", nil, "", http.StatusBadRequest))
			api.On("GetChannelByNameForTeamName", "theteam", "notthechannel", false).Return((*model.Channel)(nil), model.NewAppError("foo", "bar", nil, "", http.StatusBadRequest))
			api.On("GetChannelByNameForTeamName", "theteam", "thechannel", false).Run(func(args mock.Arguments) {
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, tc.CreatePostError)
			}).Return(&model.Channel{
				Id:     "thechannelid",
				TeamId: "theteamid",
			}, (*model.AppError)(nil))

			p := Plugin{}
			p.updateConfig(func(conf *config) {
				conf.Secret = tc.Configuration.Secret
			})
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)
			p.instanceStore = p.getMockInstanceStoreKV(1)
			p.initializeRouter()

			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, tc.Request)
			assert.Equal(t, tc.ExpectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestSetupAutolink(t *testing.T) {
	mockAPI := &plugintest.API{}
	dummyInstanceStore := new(mockInstanceStore)
	mockPluginClient := pluginapi.NewClient(mockAPI, nil)
	p := &Plugin{
		client:        mockPluginClient,
		instanceStore: dummyInstanceStore,
	}

	tests := []struct {
		name         string
		setup        func()
		InstanceType InstanceType
	}{
		{
			name: "Missing API token or Admin email",
			setup: func() {
				mockAPI.On("LogInfo", "unable to setup autolink due to missing API Token or Admin Email").Return(nil).Times(1)
				dummyInstanceStore.On("LoadInstance", mock.Anything).Return(&serverInstance{}, nil).Times(1)

				p.updateConfig(func(c *config) {
					c.AdminAPIToken = ""
					c.AdminEmail = ""
				})
			},
			InstanceType: ServerInstanceType,
		},
		{
			name: "Unsupported instance type",
			setup: func() {
				mockAPI.On("LogInfo", "only cloud and cloud-oauth instances supported for autolink").Return(nil).Times(1)
				dummyInstanceStore.On("LoadInstance", mock.Anything).Return(&serverInstance{}, nil).Times(1)

				p.updateConfig(GetConfigSetterFunction())
			},
			InstanceType: ServerInstanceType,
		},
		{
			name: "Autolink plugin unavailable API returned error",
			setup: func() {
				mockAPI.On("LogWarn", "OnActivate: Autolink plugin unavailable. API returned error", "error", mock.Anything).Return(nil).Times(1)
				mockAPI.On("GetPluginStatus", autolinkPluginID).Return(nil, &model.AppError{Message: "error getting plugin status"}).Times(1)
				dummyInstanceStore.On("LoadInstance", mock.Anything).Return(&cloudInstance{}, nil).Times(1)

				p.updateConfig(GetConfigSetterFunction())
			},
			InstanceType: CloudInstanceType,
		},
		{
			name: "Autolink plugin not running",
			setup: func() {
				mockAPI.On("LogWarn", "OnActivate: Autolink plugin unavailable. Plugin is not running", "status", &model.PluginStatus{State: model.PluginStateNotRunning}).Return(nil).Times(1)
				mockAPI.On("GetPluginStatus", autolinkPluginID).Return(&model.PluginStatus{State: model.PluginStateNotRunning}, nil).Times(1)
				dummyInstanceStore.On("LoadInstance", mock.Anything).Return(&cloudInstance{}, nil).Times(1)

				p.updateConfig(GetConfigSetterFunction())
			},
			InstanceType: CloudInstanceType,
		},
		{
			name: "Error installing autolinks for cloud instance",
			setup: func() {
				mockAPI.On("LogInfo", "could not install autolinks for cloud instance", "instance", "mockBaseURL", "err", mock.Anything).Return(nil).Times(1)
				mockAPI.On("GetPluginStatus", autolinkPluginID).Return(&model.PluginStatus{State: model.PluginStateRunning}, nil).Times(1)
				dummyInstanceStore.On("LoadInstance", mock.Anything).Return(
					&cloudInstance{
						InstanceCommon: &InstanceCommon{
							Plugin: p,
						},
						AtlassianSecurityContext: &AtlassianSecurityContext{
							BaseURL:      MockBaseURL,
							Key:          MockASCKey,
							ClientKey:    MockASCClientKey,
							SharedSecret: MockASCSharedSecret,
						},
					}, nil).Times(1)

				p.updateConfig(GetConfigSetterFunction())
			},
			InstanceType: CloudInstanceType,
		},
		{
			name: "Error installing autolinks for cloud-oauth instance",
			setup: func() {
				mockAPI.On("LogInfo", "could not install autolinks for cloud-oauth instance", "instance", "mockBaseURL", "err", mock.Anything).Return(nil).Times(1)
				mockAPI.On("GetPluginStatus", autolinkPluginID).Return(&model.PluginStatus{State: model.PluginStateRunning}, nil).Times(1)
				dummyInstanceStore.On("LoadInstance", mock.Anything).Return(
					&cloudOAuthInstance{
						InstanceCommon: &InstanceCommon{
							Plugin: p,
						},
						JiraBaseURL: MockBaseURL,
					}, nil).Times(1)

				p.updateConfig(GetConfigSetterFunction())
			},
			InstanceType: CloudOAuthInstanceType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			instances := GetInstancesWithType(tt.InstanceType)

			p.SetupAutolink(instances)

			mockAPI.AssertExpectations(t)
			dummyInstanceStore.AssertExpectations(t)
		})
	}
}

func GetConfigSetterFunction() func(*config) {
	return func(c *config) {
		c.AdminAPIToken = MockAPIToken
		c.AdminEmail = MockAdminEmail
	}
}

func GetInstancesWithType(instanceType InstanceType) *Instances {
	return NewInstances(&InstanceCommon{
		InstanceID: MockInstanceID,
		Type:       instanceType,
	})
}
