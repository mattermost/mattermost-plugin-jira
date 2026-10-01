// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func TestCloudOAuthMigration(t *testing.T) {
	jiraCloudURL := "https://mmtest.atlassian.net"
	mmUserID := "someuserid"

	storeConnectInstance := func(t *testing.T, p *Plugin) {
		err := p.instanceStore.StoreInstance(&cloudInstance{
			InstanceCommon:              newInstanceCommon(p, CloudInstanceType, types.ID(jiraCloudURL)),
			RawAtlassianSecurityContext: `{"baseUrl":"` + jiraCloudURL + `"}`,
		})
		require.NoError(t, err)
	}

	installOAuthInstance := func(t *testing.T, p *Plugin) {
		jiraURL, oauthInstance, err := p.installCloudOAuthInstance(jiraCloudURL)
		require.NoError(t, err)
		require.NotEmpty(t, jiraURL)
		require.NotNil(t, oauthInstance)
	}

	serveAccessibleResources := func(t *testing.T) {
		fakeJiraResourcesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(JiraAccessibleResources{{ID: "someid", URL: jiraCloudURL}})
		}))
		t.Cleanup(fakeJiraResourcesServer.Close)

		oldResourcesURL := jiraOAuthAccessibleResourcesURL
		t.Cleanup(func() { jiraOAuthAccessibleResourcesURL = oldResourcesURL })
		jiraOAuthAccessibleResourcesURL = fakeJiraResourcesServer.URL
	}

	oauthConnection := func() *Connection {
		return &Connection{
			User:          jira.User{},
			PluginVersion: "4.0.1",
			OAuth2Token:   &oauth2.Token{RefreshToken: "somerefreshtoken", AccessToken: "someaccesstoken"},
			Settings:      &ConnectionSettings{},
		}
	}

	for name, tc := range map[string]struct {
		connection            *Connection
		connectedInstanceType InstanceType
		setup                 func(t *testing.T, p *Plugin) (instanceID string)
		runAssertions         func(t *testing.T, p *Plugin, instanceID string)
	}{
		"no installed instance": {
			setup: func(t *testing.T, p *Plugin) string { return "" },
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), jiraCloudURL)
				require.Error(t, err)

				_, _, _, err = p.getClient(types.ID(jiraCloudURL), types.ID(mmUserID))
				require.Error(t, err)
				require.Equal(t, "https://mmtest.atlassian.net: jira_instance_b5f8e96862ed24709919a73271ae8851: not found", err.Error())
			},
		},
		"Connect instance installed. user is connected. should ask for a reinstall": {
			connection:            &Connection{User: jira.User{AccountID: "someaccountid"}, Settings: &ConnectionSettings{}},
			connectedInstanceType: CloudInstanceType,
			setup: func(t *testing.T, p *Plugin) string {
				storeConnectInstance(t, p)
				return jiraCloudURL
			},
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				_, _, _, err := p.getClient(types.ID(jiraCloudURL), types.ID(mmUserID))
				require.ErrorIs(t, err, errConnectInstanceUnsupported)
			},
		},
		"OAuth installed over Connect instance. should replace it": {
			setup: func(t *testing.T, p *Plugin) string {
				storeConnectInstance(t, p)
				installOAuthInstance(t, p)
				return jiraCloudURL
			},
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				instance, err := p.instanceStore.LoadInstance(types.ID(jiraCloudURL))
				require.NoError(t, err)
				require.IsType(t, &cloudOAuthInstance{}, instance)
			},
		},
		"oauth installed. user is not connected. should ask to connect": {
			setup: func(t *testing.T, p *Plugin) string {
				installOAuthInstance(t, p)
				return jiraCloudURL
			},
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), jiraCloudURL)
				require.Error(t, err)

				_, _, _, err = p.getClient(types.ID(jiraCloudURL), types.ID(mmUserID))
				require.ErrorContains(t, err, "your Jira account is not connected, please use `/jira connect`")
			},
		},
		"oauth installed. user is connected to oauth. should return client for oauth": {
			connection:            oauthConnection(),
			connectedInstanceType: CloudOAuthInstanceType,
			setup: func(t *testing.T, p *Plugin) string {
				installOAuthInstance(t, p)
				return jiraCloudURL
			},
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				serveAccessibleResources(t)
				c, i, conn, err := p.getClient(types.ID(jiraCloudURL), types.ID(mmUserID))
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotNil(t, i)
				require.NotNil(t, conn)
			},
		},
		"oauth installed twice. user is connected to oauth. should return client for oauth": {
			connection:            oauthConnection(),
			connectedInstanceType: CloudOAuthInstanceType,
			setup: func(t *testing.T, p *Plugin) string {
				installOAuthInstance(t, p)
				installOAuthInstance(t, p)
				return jiraCloudURL
			},
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				serveAccessibleResources(t)
				c, i, conn, err := p.getClient(types.ID(jiraCloudURL), types.ID(mmUserID))
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotNil(t, i)
				require.NotNil(t, conn)
			},
		},
		"Jira Server instance installed. user is not connected. should return nil user instance": {
			setup: func(t *testing.T, p *Plugin) string {
				fakeJiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_ = json.NewEncoder(w).Encode(utils.JiraStatus{State: "RUNNING"})
				}))
				defer fakeJiraServer.Close()

				_, _, err := p.installServerInstance(fakeJiraServer.URL)
				require.NoError(t, err)
				return fakeJiraServer.URL
			},
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), instanceID)
				require.Error(t, err)

				_, _, _, err = p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.Error(t, err)
				require.Equal(t, "failed to get a Jira client for : no access token, please use /jira connect", err.Error())
			},
		},
		"Jira Server instance installed. user is connected. should return client for Jira Server": {
			connection: &Connection{
				User:               jira.User{},
				PluginVersion:      "4.0.1",
				Oauth1AccessToken:  "jiraserveraccesstoken",
				Oauth1AccessSecret: "jiraserveraccesssecret",
				Settings:           &ConnectionSettings{},
			},
			connectedInstanceType: ServerInstanceType,
			setup: func(t *testing.T, p *Plugin) string {
				fakeJiraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_ = json.NewEncoder(w).Encode(utils.JiraStatus{State: "RUNNING"})
				}))
				defer fakeJiraServer.Close()

				_, _, err := p.installServerInstance(fakeJiraServer.URL)
				require.NoError(t, err)
				return fakeJiraServer.URL
			},
			runAssertions: func(t *testing.T, p *Plugin, instanceID string) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), instanceID)
				require.NoError(t, err)

				_, _, _, err = p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := &Plugin{}
			p.updateConfig(func(conf *config) {
				conf.mattermostSiteURL = mattermostSiteURL
			})

			api := &plugintest.API{}
			p.SetAPI(api)

			testStore := makeTestKVStore(api, testKVStore{})
			require.NotNil(t, testStore)

			store := NewStore(p)
			p.instanceStore = store
			p.userStore = store
			p.secretsStore = store
			p.otsStore = store
			p.client = pluginapi.NewClient(p.API, p.Driver)
			p.enterpriseChecker = &mockEnterpriseChecker{false}

			tempDir, err := os.MkdirTemp("", "sampledir")
			require.NoError(t, err)
			err = os.Mkdir(tempDir+"/assets", 0777)
			require.NoError(t, err)
			err = os.WriteFile(tempDir+"/assets/icon.svg", []byte("<svg/>"), 0600)
			require.NoError(t, err)

			api.On("GetBundlePath").Return(tempDir, nil)
			api.On("LogDebug", mock.AnythingOfType("string")).Return()
			api.On("UnregisterCommand", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			api.On("RegisterCommand", mock.Anything).Return(nil)
			api.On("PublishWebSocketEvent", mock.AnythingOfType("string"), mock.Anything, mock.Anything)

			installedInstanceID := tc.setup(t, p)

			if tc.connection != nil {
				err = p.userStore.StoreConnection(types.ID(installedInstanceID), types.ID(mmUserID), tc.connection)
				require.NoError(t, err)

				connectedInstances := NewInstances(newInstanceCommon(p, tc.connectedInstanceType, types.ID(installedInstanceID)))
				storeUser := User{ConnectedInstances: connectedInstances, MattermostUserID: types.ID(mmUserID)}
				err = p.userStore.StoreUser(&storeUser)
				require.NoError(t, err)
			}

			tc.runAssertions(t, p, installedInstanceID)
		})
	}
}

type mockEnterpriseChecker struct {
	hasEnterpriseFeatures bool
}

func (mec *mockEnterpriseChecker) HasEnterpriseFeatures() bool {
	return mec.hasEnterpriseFeatures
}
