package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

func validRequestBody() io.ReadCloser {
	if f, err := os.Open("testdata/webhook_issue_created.json"); err != nil {
		panic(err)
	} else {
		return f
	}
}

func TestPlugin(t *testing.T) {
	f, err := os.Open("testdata/webhook_issue_created.json")
	require.NoError(t, err)
	defer f.Close()
	var webhook Webhook
	require.NoError(t, json.NewDecoder(f).Decode(&webhook))
	expectedAttachment, err := webhook.SlackAttachment()
	require.NoError(t, err)

	validConfiguration := Configuration{
		Enabled:  true,
		Secret:   "thesecret",
		UserName: "theuser",
	}

	for name, tc := range map[string]struct {
		Configuration      Configuration
		ConfigurationError error
		Request            *http.Request
		CreatePostError    *model.AppError
		ExpectedStatusCode int
	}{
		"NoConfiguration": {
			Configuration:      Configuration{},
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusForbidden,
		},
		"ConfigurationError": {
			ConfigurationError: fmt.Errorf("foo"),
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusForbidden,
		},
		"NoUserConfiguration": {
			Configuration: Configuration{
				Enabled: true,
				Secret:  "thesecret",
			},
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&channel=thechannel&secret=thesecret", validRequestBody()),
			ExpectedStatusCode: http.StatusForbidden,
		},
		"NoChannel": {
			Configuration:      validConfiguration,
			Request:            httptest.NewRequest("POST", "/webhook?team=theteam&secret=thesecret", validRequestBody()),
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
			ExpectedStatusCode: http.StatusOK,
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
			Configuration: Configuration{
				Enabled:  true,
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

			api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.Configuration")).Return(func(dest interface{}) error {
				*dest.(*Configuration) = tc.Configuration
				return tc.ConfigurationError
			})

			api.On("GetUserByUsername", "theuser").Return(&model.User{
				Id: "theuserid",
			}, (*model.AppError)(nil))
			api.On("GetUserByUsername", "nottheuser").Return((*model.User)(nil), model.NewAppError("foo", "bar", nil, "", http.StatusBadRequest))

			api.On("GetTeamByName", "theteam").Return(&model.Team{
				Id: "theteamid",
			}, (*model.AppError)(nil))
			api.On("GetTeamByName", "nottheteam").Return((*model.Team)(nil), model.NewAppError("foo", "bar", nil, "", http.StatusBadRequest))

			api.On("GetChannelByName", "thechannel", "theteamid").Run(func(args mock.Arguments) {
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(func(post *model.Post) (*model.Post, *model.AppError) {
					assert.Equal(t, post.ChannelId, "thechannelid")
					assert.Equal(t, post.Props["attachments"], []*model.SlackAttachment{expectedAttachment})
					return &model.Post{}, tc.CreatePostError
				})
			}).Return(&model.Channel{
				Id:     "thechannelid",
				TeamId: "theteamid",
			}, (*model.AppError)(nil))
			api.On("GetChannelByName", "notthechannel", "theteamid").Return((*model.Channel)(nil), model.NewAppError("foo", "bar", nil, "", http.StatusBadRequest))

			p := Plugin{}
			p.OnActivate(api)

			w := httptest.NewRecorder()
			p.ServeHTTP(w, tc.Request)
			assert.Equal(t, tc.ExpectedStatusCode, w.Result().StatusCode)
		})
	}
}
