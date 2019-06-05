// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
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
		Secret:   "thesecret",
		UserName: "theuser",
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
		"NoUserConfiguration": {
			Configuration: TestConfiguration{
				Secret: "thesecret",
			},
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
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", ioutil.NopCloser(bytes.NewBufferString("foo"))),
			ExpectedStatusCode: http.StatusBadRequest,
		},
		"UnknownJSONPayload": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", ioutil.NopCloser(bytes.NewBufferString("{}"))),
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
		"InvalidUser": {
			Configuration: TestConfiguration{
				Secret:   "thesecret",
				UserName: "nottheuser",
			},
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", validRequestBody()),
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

			api.On("LogDebug",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)

			api.On("KVGet", mock.AnythingOfTypeArgument("string")).Return(make([]byte, 0), (*model.AppError)(nil))
			api.On("GetDirectChannel", mock.AnythingOfTypeArgument("string"), mock.AnythingOfTypeArgument("string")).Return(
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
				conf.UserName = tc.Configuration.UserName
			})
			p.SetAPI(api)
			p.currentInstanceStore = mockCurrentInstanceStore{}

			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, tc.Request)
			assert.Equal(t, tc.ExpectedStatusCode, w.Result().StatusCode)
		})
	}
}
