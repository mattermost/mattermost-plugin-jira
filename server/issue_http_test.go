package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShareIssuePubliclyAuthentication(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)
	api.On("LogWarn", mockAnythingOfTypeBatch("string", 10)...).Return(nil)
	api.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(&model.Post{})
	api.On("GetPost", "attacker-post").Return(&model.Post{UserId: "attacker"}, (*model.AppError)(nil))

	p := &Plugin{}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.initializeRouter()
	p.updateConfig(func(conf *config) {
		conf.botUserID = "bot-user"
	})

	t.Run("missing Mattermost user header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/share-issue-publicly", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		p.ServeHTTP(&plugin.Context{}, w, req)
		require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})

	t.Run("post not authored by jira bot", func(t *testing.T) {
		payload := model.PostActionIntegrationRequest{
			UserId: "victim",
			PostId: "attacker-post",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v2/share-issue-publicly", bytes.NewReader(body))
		req.Header.Set("Mattermost-User-ID", "victim")

		w := httptest.NewRecorder()
		p.ServeHTTP(&plugin.Context{}, w, req)
		require.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}
