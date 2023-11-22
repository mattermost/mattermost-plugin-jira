package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCloudOAuthMigration(t *testing.T) {
	instanceID := "https://mmtest.atlassian.net"
	mmUserID := "someuserid"

	for name, tc := range map[string]struct {
		connection            *Connection
		connectedInstanceType InstanceType
		setup                 func(p *Plugin, api *plugintest.API)
		runAssertions         func(p *Plugin, api *plugintest.API)
	}{
		"no installed instance": {
			connection: nil,
			setup:      func(p *Plugin, api *plugintest.API) {},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), instanceID)
				require.Error(t, err)

				_, _, _, err = p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.Error(t, err)
				require.Equal(t, "https://mmtest.atlassian.net: jira_instance_b5f8e96862ed24709919a73271ae8851: not found", err.Error())
			},
		},
		"JWT instance installed. user is not connected. should return nil user instance": {
			connection: nil,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: new Jira Cloud instance: https://mmtest.atlassian.net as jira_instance_b5f8e96862ed24709919a73271ae8851").Twice().Return(nil)
				_, err := p.installInactiveCloudInstance(instanceID, mmUserID)
				require.NoError(t, err)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), instanceID)
				require.Error(t, err)

				_, _, _, err = p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err) // this should return an error but doesn't, since we don't return an error if it doesn't exist in the kv store. doesn't seem to cause any issues with the plugin so maybe its okay. LoadUserInstance takes precedence when the plugin looks up the user's connection
			},
		},
		"JWT instance installed. user is connected. should return valid JWT client": {
			connection: &Connection{
				User:               jira.User{},
				PluginVersion:      "4.0.1",
				Oauth1AccessToken:  "someaccesstoken",
				Oauth1AccessSecret: "someaccesssecret",
				OAuth2Token:        nil,
				Settings:           &ConnectionSettings{},
				DefaultProjectKey:  "",
				MattermostUserID:   types.ID(mmUserID),
			},
			connectedInstanceType: CloudInstanceType,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: new Jira Cloud instance: https://mmtest.atlassian.net as jira_instance_b5f8e96862ed24709919a73271ae8851").Return(nil)
				api.On("LogDebug", "Stored: user someuserid key:user_daa0ef689b843fada63e9f383fce33e1: connected to:[\"https://mmtest.atlassian.net\"]").Return(nil)
				api.On("LogDebug", "Stored: connection, keys:\n\t6d03c97fdd1dee73b64caeca04e3e0d6 (someuserid): \n\t0f1a5629834c263cd6a3d59ce216c1f5 (): someuserid").Return(nil)
				_, err := p.installInactiveCloudInstance(instanceID, mmUserID)
				require.NoError(t, err)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				_, i, err := p.LoadUserInstance(types.ID(mmUserID), instanceID)
				require.NoError(t, err)
				require.Equal(t, CloudInstanceType, i.Common().Type)

				_, _, _, err = p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err)
			},
		},
		"migrated JWT to oauth. user is not connected. should return nil user instance": {
			connection: nil,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: new Jira Cloud instance: https://mmtest.atlassian.net as jira_instance_b5f8e96862ed24709919a73271ae8851").Return(nil)
				api.On("LogDebug", "Stored: user someuserid key:user_daa0ef689b843fada63e9f383fce33e1: connected to:[\"https://mmtest.atlassian.net\"]").Return(nil)
				api.On("LogDebug", "Stored: connection, keys:\n\t6d03c97fdd1dee73b64caeca04e3e0d6 (someuserid): \n\t0f1a5629834c263cd6a3d59ce216c1f5 (): someuserid").Return(nil)

				_, err := p.installInactiveCloudInstance(instanceID, mmUserID)
				require.NoError(t, err)

				api.On("LogDebug", "Installing cloud-oauth over existing cloud JWT instance. Carrying over existing saved JWT instance.").Return(nil)

				jiraURL, oauthInstance, err := p.installCloudOAuthInstance(instanceID)
				require.NoError(t, err)
				require.NotEmpty(t, jiraURL)
				require.NotNil(t, oauthInstance)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), instanceID)
				require.Error(t, err)

				api.On("LogDebug", "Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil").Return(nil)
				_, _, _, err = p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err) // this should return an error but doesn't, since we don't return an error if it doesn't exist in the kv store. doesn't seem to cause any issues with the plugin so maybe its okay. LoadUserInstance takes precedence when the plugin looks up the user's connection
			},
		},
		"migrated JWT to oauth. user is connected to JWT but not oauth. should return client for JWT": {
			connection: &Connection{
				User:               jira.User{},
				PluginVersion:      "4.0.1",
				Oauth1AccessToken:  "someaccesstoken",
				Oauth1AccessSecret: "someaccesssecret",
				OAuth2Token:        nil,
				Settings:           &ConnectionSettings{},
				DefaultProjectKey:  "",
				MattermostUserID:   types.ID(mmUserID),
			},
			connectedInstanceType: CloudInstanceType,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: new Jira Cloud instance: https://mmtest.atlassian.net as jira_instance_b5f8e96862ed24709919a73271ae8851").Return(nil)
				api.On("LogDebug", "Stored: user someuserid key:user_daa0ef689b843fada63e9f383fce33e1: connected to:[\"https://mmtest.atlassian.net\"]").Return(nil)
				api.On("LogDebug", "Stored: connection, keys:\n\t6d03c97fdd1dee73b64caeca04e3e0d6 (someuserid): \n\t0f1a5629834c263cd6a3d59ce216c1f5 (): someuserid").Return(nil)

				_, err := p.installInactiveCloudInstance(instanceID, mmUserID)
				require.NoError(t, err)

				api.On("LogDebug", "Installing cloud-oauth over existing cloud JWT instance. Carrying over existing saved JWT instance.").Return(nil)

				jiraURL, oauthInstance, err := p.installCloudOAuthInstance(instanceID)
				require.NoError(t, err)
				require.NotEmpty(t, jiraURL)
				require.NotNil(t, oauthInstance)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil").Return(nil)
				c, i, conn, err := p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotNil(t, i)
				require.NotNil(t, conn)
			},
		},
		"migrated JWT to oauth. user is connected to JWT and oauth. should return client for oauth": {
			connection: &Connection{
				User:               jira.User{},
				PluginVersion:      "4.0.1",
				Oauth1AccessToken:  "someaccesstoken",
				Oauth1AccessSecret: "someaccesssecret",
				OAuth2Token:        &oauth2.Token{RefreshToken: "somerefreshtoken", AccessToken: "someaccesstoken"},
				Settings:           &ConnectionSettings{},
				DefaultProjectKey:  "",
				MattermostUserID:   types.ID(mmUserID),
			},
			connectedInstanceType: CloudOAuthInstanceType,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: new Jira Cloud instance: https://mmtest.atlassian.net as jira_instance_b5f8e96862ed24709919a73271ae8851").Return(nil)
				api.On("LogDebug", "Stored: user someuserid key:user_daa0ef689b843fada63e9f383fce33e1: connected to:[\"https://mmtest.atlassian.net\"]").Return(nil)
				api.On("LogDebug", "Stored: connection, keys:\n\t6d03c97fdd1dee73b64caeca04e3e0d6 (someuserid): \n\t0f1a5629834c263cd6a3d59ce216c1f5 (): someuserid").Return(nil)

				_, err := p.installInactiveCloudInstance(instanceID, mmUserID)
				require.NoError(t, err)

				api.On("LogDebug", "Installing cloud-oauth over existing cloud JWT instance. Carrying over existing saved JWT instance.").Return(nil)
				jiraURL, oauthInstance, err := p.installCloudOAuthInstance(instanceID)
				require.NoError(t, err)
				require.NotEmpty(t, jiraURL)
				require.NotNil(t, oauthInstance)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				fakeJiraResourcesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					accessibleResources := JiraAccessibleResources{{
						ID: "someid",
					}}
					json.NewEncoder(w).Encode(accessibleResources)
				}))

				defer fakeJiraResourcesServer.Close()

				oldResourcesURL := jiraOAuthAccessibleResourcesURL
				defer func() {
					jiraOAuthAccessibleResourcesURL = oldResourcesURL
				}()

				jiraOAuthAccessibleResourcesURL = fakeJiraResourcesServer.URL

				// Times(0) means this will not get logged
				api.On("LogDebug", "Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil").Times(0).Return(nil)
				c, i, conn, err := p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotNil(t, i)
				require.NotNil(t, conn)
			},
		},
		"migrated JWT to oauth. user is connected to oauth but not JWT. should return client for oauth": {
			connection: &Connection{
				User:               jira.User{},
				PluginVersion:      "4.0.1",
				Oauth1AccessToken:  "",
				Oauth1AccessSecret: "",
				OAuth2Token:        &oauth2.Token{RefreshToken: "somerefreshtoken", AccessToken: "someaccesstoken"},
				Settings:           &ConnectionSettings{},
				DefaultProjectKey:  "",
				MattermostUserID:   types.ID(mmUserID),
			},
			connectedInstanceType: CloudOAuthInstanceType,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: new Jira Cloud instance: https://mmtest.atlassian.net as jira_instance_b5f8e96862ed24709919a73271ae8851").Return(nil)
				api.On("LogDebug", "Stored: user someuserid key:user_daa0ef689b843fada63e9f383fce33e1: connected to:[\"https://mmtest.atlassian.net\"]").Return(nil)
				api.On("LogDebug", "Stored: connection, keys:\n\t6d03c97fdd1dee73b64caeca04e3e0d6 (someuserid): \n\t0f1a5629834c263cd6a3d59ce216c1f5 (): someuserid").Return(nil)

				_, err := p.installInactiveCloudInstance(instanceID, mmUserID)
				require.NoError(t, err)

				api.On("LogDebug", "Installing cloud-oauth over existing cloud JWT instance. Carrying over existing saved JWT instance.").Return(nil)
				jiraURL, oauthInstance, err := p.installCloudOAuthInstance(instanceID)
				require.NoError(t, err)
				require.NotEmpty(t, jiraURL)
				require.NotNil(t, oauthInstance)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				fakeJiraResourcesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					accessibleResources := JiraAccessibleResources{{
						ID: "someid",
					}}
					json.NewEncoder(w).Encode(accessibleResources)
				}))

				defer fakeJiraResourcesServer.Close()

				oldResourcesURL := jiraOAuthAccessibleResourcesURL
				defer func() {
					jiraOAuthAccessibleResourcesURL = oldResourcesURL
				}()

				jiraOAuthAccessibleResourcesURL = fakeJiraResourcesServer.URL

				// Times(0) means this will not get logged
				api.On("LogDebug", "Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil").Times(0).Return(nil)
				c, i, conn, err := p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotNil(t, i)
				require.NotNil(t, conn)
			},
		},
		"oauth installed without JWT instance. user is not connected. should return nil user instance": {
			connection: nil,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: user someuserid key:user_daa0ef689b843fada63e9f383fce33e1: connected to:[\"https://mmtest.atlassian.net\"]").Return(nil)
				api.On("LogDebug", "Stored: connection, keys:\n\t6d03c97fdd1dee73b64caeca04e3e0d6 (someuserid): \n\t0f1a5629834c263cd6a3d59ce216c1f5 (): someuserid").Return(nil)

				api.On("LogDebug", "Installing new cloud-oauth instance. There exists no previous JWT instance to carry over.").Return(nil)
				jiraURL, oauthInstance, err := p.installCloudOAuthInstance(instanceID)
				require.NoError(t, err)
				require.NotEmpty(t, jiraURL)
				require.NotNil(t, oauthInstance)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				_, _, err := p.LoadUserInstance(types.ID(mmUserID), instanceID)
				require.Error(t, err)

				// Times(0) means this will not get logged
				api.On("LogDebug", "Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil").Times(0).Return(nil)
				_, _, _, err = p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.Error(t, err)
				require.Equal(t, "failed to get Jira client for the user : failed to create client for OAuth instance: no JWT instance found, and connection's OAuth token is missing", err.Error())
			},
		},
		"oauth installed without JWT instance. user is connected to oauth. should return client for oauth": {
			connection: &Connection{
				User:               jira.User{},
				PluginVersion:      "4.0.1",
				Oauth1AccessToken:  "",
				Oauth1AccessSecret: "",
				OAuth2Token:        &oauth2.Token{RefreshToken: "somerefreshtoken", AccessToken: "someaccesstoken"},
				Settings:           &ConnectionSettings{},
				DefaultProjectKey:  "",
				MattermostUserID:   types.ID(mmUserID),
			},
			connectedInstanceType: CloudOAuthInstanceType,
			setup: func(p *Plugin, api *plugintest.API) {
				api.On("LogDebug", "Stored: user someuserid key:user_daa0ef689b843fada63e9f383fce33e1: connected to:[\"https://mmtest.atlassian.net\"]").Return(nil)
				api.On("LogDebug", "Stored: connection, keys:\n\t6d03c97fdd1dee73b64caeca04e3e0d6 (someuserid): \n\t0f1a5629834c263cd6a3d59ce216c1f5 (): someuserid").Return(nil)

				api.On("LogDebug", "Installing new cloud-oauth instance. There exists no previous JWT instance to carry over.").Return(nil)
				jiraURL, oauthInstance, err := p.installCloudOAuthInstance(instanceID)
				require.NoError(t, err)
				require.NotEmpty(t, jiraURL)
				require.NotNil(t, oauthInstance)
			},
			runAssertions: func(p *Plugin, api *plugintest.API) {
				fakeJiraResourcesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					accessibleResources := JiraAccessibleResources{{
						ID: "someid",
					}}
					json.NewEncoder(w).Encode(accessibleResources)
				}))

				defer fakeJiraResourcesServer.Close()

				oldResourcesURL := jiraOAuthAccessibleResourcesURL
				defer func() {
					jiraOAuthAccessibleResourcesURL = oldResourcesURL
				}()

				jiraOAuthAccessibleResourcesURL = fakeJiraResourcesServer.URL

				// Times(0) means this will not get logged
				api.On("LogDebug", "Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil").Times(0).Return(nil)
				c, i, conn, err := p.getClient(types.ID(instanceID), types.ID(mmUserID))
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotNil(t, i)
				require.NotNil(t, conn)
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
			p.client = pluginapi.NewClient(p.API, p.Driver)

			testStore := makeTestKVStore(api, testKVStore{})
			require.NotNil(t, testStore)

			store := NewStore(p)
			p.instanceStore = store
			p.userStore = store
			p.secretsStore = store
			p.otsStore = store
			p.client = pluginapi.NewClient(p.API, p.Driver)

			eCheck := &mockEnterpriseChecker{false}
			p.enterpriseChecker = eCheck

			tempDir, err := os.MkdirTemp("", "sampledir")
			require.NoError(t, err)

			err = os.Mkdir(tempDir+"/assets", 0777)
			require.NoError(t, err)

			err = os.WriteFile(tempDir+"/assets/icon.svg", []byte("<svg/>"), 0666)
			require.NoError(t, err)

			api.On("GetBundlePath").Return(tempDir, nil)

			api.On("UnregisterCommand", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			api.On("RegisterCommand", mock.Anything).Return(nil)
			api.On("PublishWebSocketEvent", mock.AnythingOfTypeArgument("string"), mock.Anything, mock.Anything)

			tc.setup(p, api)

			if tc.connection != nil {
				err = p.userStore.StoreConnection(types.ID(instanceID), types.ID(mmUserID), tc.connection)
				require.NoError(t, err)

				connectedInstances := NewInstances(newInstanceCommon(p, tc.connectedInstanceType, types.ID(instanceID)))
				storeUser := User{ConnectedInstances: connectedInstances, MattermostUserID: types.ID(mmUserID)}
				p.userStore.StoreUser(&storeUser)
			}

			tc.runAssertions(p, api)
		})
	}
}

type mockEnterpriseChecker struct {
	hasEnterpriseFeatures bool
}

func (mec *mockEnterpriseChecker) HasEnterpriseFeatures() bool {
	return mec.hasEnterpriseFeatures
}
