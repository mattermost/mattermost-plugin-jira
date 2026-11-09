// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func TestExpandIssueCreatedWithoutCreator(t *testing.T) {
	newWebhook := func() *JiraWebhook {
		return &JiraWebhook{
			WebhookEvent: issueCreated,
			Issue:        jira.Issue{ID: "10001", Key: "PROJ-1", Fields: &jira.IssueFields{}},
		}
	}

	setup := func(t *testing.T, jiraBaseURL string) (*Plugin, *plugintest.API, *mockUserStoreForTokenExpiry, types.ID) {
		instance := &cloudOAuthInstance{
			InstanceCommon: &InstanceCommon{InstanceID: types.ID(jiraBaseURL), Type: CloudOAuthInstanceType},
			JiraBaseURL:    jiraBaseURL,
		}
		instanceStore := &mockInstanceStore{}
		instanceStore.On("LoadInstance", instance.GetID()).Return(instance, nil)

		api := &plugintest.API{}
		userStore := &mockUserStoreForTokenExpiry{}
		p := setupTestPlugin(api)
		p.userStore = userStore
		p.instanceStore = instanceStore
		return p, api, userStore, instance.GetID()
	}

	setAdminToken := func(p *Plugin) {
		tokenJSON, _ := json.Marshal("test-api-token")
		p.updateConfig(func(conf *config) {
			conf.AdminAPIToken = string(tokenJSON)
			conf.AdminEmail = "admin@example.com"
			conf.EncryptionKey = ""
		})
	}

	t.Run("returns an error when no admin token is configured", func(t *testing.T) {
		p, _, userStore, instanceID := setup(t, "https://mmtest.atlassian.net")

		err := newWebhook().expandIssue(p, instanceID)
		require.ErrorContains(t, err, "issue creator is missing")
		userStore.AssertNotCalled(t, "LoadMattermostUserID", mock.Anything, mock.Anything)
	})

	t.Run("fetches the issue with the admin token", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/api/2/issue/PROJ-1", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10001","key":"PROJ-1","fields":{"summary":"fetched summary"}}`))
		}))
		defer ts.Close()

		p, _, _, instanceID := setup(t, ts.URL)
		setAdminToken(p)

		jwh := newWebhook()
		require.NoError(t, jwh.expandIssue(p, instanceID))
		require.NotNil(t, jwh.Issue.Fields)
		assert.Equal(t, "fetched summary", jwh.Issue.Fields.Summary)
	})

	t.Run("returns an error when the fetched issue has no fields", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"10001","key":"PROJ-1"}`))
		}))
		defer ts.Close()

		p, _, _, instanceID := setup(t, ts.URL)
		setAdminToken(p)

		jwh := newWebhook()
		require.ErrorContains(t, jwh.expandIssue(p, instanceID), "no fields")
		require.NotNil(t, jwh.Issue.Fields)
	})
}
