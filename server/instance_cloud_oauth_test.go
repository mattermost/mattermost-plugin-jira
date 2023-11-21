package main

import (
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	cloudInstanceData = iota
	cloudOAuthInstanceData
	cloudOAuthInstanceDataWithJWTInstance
	emptyData
)

func SetData(dataType int, api *plugintest.API) {
	switch dataType {
	case cloudInstanceData:
		makeTestKVStore(api, testKVStore{
			hashkey("jira_instance_", "https://brightscout-dev.atlassian.net"): []byte(`{"InstanceID":"","Alias":"","Type":"cloud","IsV2Legacy":false,"SetupWizardUserID":"","Installed":true,"RawAtlassianSecurityContext":"{\"key\":\"mattermost_b7_122_176_55_206_ngrok_free_app\",\"clientKey\":\"mockClientKey\",\"oauthClientId\":\"mockOAuthCLientID\",\"publicKey\":\"mockPublicKey\",\"sharedSecret\":\"mockSharedSecret\",\"serverVersion\":\"100242\",\"pluginsVersion\":\"1001.0.0-SNAPSHOT\",\"baseUrl\":\"https://brightscout-dev.atlassian.net\",\"displayUrl\":\"https://brightscout-dev.atlassian.net\",\"productType\":\"jira\",\"description\":\"Atlassian JIRA at https://brightscout-dev.atlassian.net \",\"eventType\":\"installed\",\"displayUrlServicedeskHelpCenter\":\"https://brightscout-dev.atlassian.net\"}"}`),
		})
	case cloudOAuthInstanceData:
		makeTestKVStore(api, testKVStore{
			hashkey("jira_instance_", "https://brightscout-dev.atlassian.net"): []byte(`{"InstanceID":"","Alias":"","Type":"cloud-oauth","IsV2Legacy":false,"SetupWizardUserID":"","MattermostKey":"mattermost_186_125_63_102_78_ngrok_free_app","JiraResourceID":"","JiraClientID":"","JiraClientSecret":"","JiraBaseURL":"https://brightscout-dev.atlassian.net","CodeVerifier":"mockCodeVerifier","CodeChallenge":"mockCodeChallenge","JWTInstance":null}`),
		})
	case cloudOAuthInstanceDataWithJWTInstance:
		makeTestKVStore(api, testKVStore{
			hashkey("jira_instance_", "https://brightscout-dev.atlassian.net"): []byte(`{"InstanceID":"","Alias":"","Type":"cloud-oauth","IsV2Legacy":false,"SetupWizardUserID":"","MattermostKey":"mattermost_186_125_63_102_78_ngrok_free_app","JiraResourceID":"","JiraClientID":"","JiraClientSecret":"","JiraBaseURL":"https://brightscout-dev.atlassian.net","CodeVerifier":"mockCodeVerifier","CodeChallenge":"mockCodeChallenge","JWTInstance":{"InstanceID":"","Alias":"","Type":"cloud","IsV2Legacy":false,"SetupWizardUserID":"","Installed":true,"RawAtlassianSecurityContext":"{\"key\":\"mattermost_b7_122_176_55_206_ngrok_free_app\",\"clientKey\":\"mockClientKey\",\"oauthClientId\":\"mockOAuthCLientID\",\"publicKey\":\"mockPublicKey\",\"sharedSecret\":\"mockSharedSecret\",\"serverVersion\":\"100242\",\"pluginsVersion\":\"1001.0.0-SNAPSHOT\",\"baseUrl\":\"https://brightscout-dev.atlassian.net\",\"displayUrl\":\"https://brightscout-dev.atlassian.net\",\"productType\":\"jira\",\"description\":\"Atlassian JIRA at https://brightscout-dev.atlassian.net \",\"eventType\":\"installed\",\"displayUrlServicedeskHelpCenter\":\"https://brightscout-dev.atlassian.net\"}"}}
			`),
		})
	default:
		makeTestKVStore(api, testKVStore{
			hashkey("jira_instance_", "https://brightscout-dev.atlassian.net"): []byte(nil),
		})
	}
}

func TestInstallCloudOAuthInstance(t *testing.T) {
	for name, test := range map[string]struct {
		JiraURL                        string
		SetupAPI                       func(*plugintest.API)
		InitializeKVStore              func()
		HasPreviouslyInstalledInstance bool
	}{
		"InstallCloudOAuthInstance: successfully carried previous JWT instance to oauth instance": {
			JiraURL:                        "https://brightscout-dev.atlassian.net",
			HasPreviouslyInstalledInstance: true,
			SetupAPI: func(api *plugintest.API) {
				api.On("LogDebug", "Installing cloud-oauth over existing cloud JWT instance. Carrying over existing saved JWT instance.").Once()
				SetData(cloudInstanceData, api)
			},
		},
		"InstallCloudOAuthInstance: successfully carried JWT instance from previous oauth instance to new oauth instance": {
			JiraURL:                        "https://brightscout-dev.atlassian.net",
			HasPreviouslyInstalledInstance: true,
			SetupAPI: func(api *plugintest.API) {
				api.On("LogDebug", "Installing cloud-oauth over existing cloud-oauth instance. Carrying over existing saved JWT instance.").Once()
				SetData(cloudOAuthInstanceDataWithJWTInstance, api)
			},
		},
		"InstallCloudOAuthInstance: no JWT instance installed previously": {
			JiraURL:                        "https://brightscout-dev.atlassian.net",
			HasPreviouslyInstalledInstance: false,
			SetupAPI: func(api *plugintest.API) {
				api.On("LogDebug", "Installing new cloud-oauth instance. There exists no previous JWT instance to carry over.").Once()
				SetData(emptyData, api)
			},
		},
		"InstallCloudOAuthInstance: no JWT instance present inside previous oauth instance": {
			JiraURL:                        "https://brightscout-dev.atlassian.net",
			HasPreviouslyInstalledInstance: false,
			SetupAPI: func(api *plugintest.API) {
				api.On("LogDebug", "Installing cloud-oauth over existing cloud-oauth instance. There exists no previous JWT instance to carry over.").Once()
				SetData(cloudOAuthInstanceData, api)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := &Plugin{}

			p.instanceStore = mockInstanceStoreForOauthMigration{plugin: p}
			p.enterpriseChecker = &mockChecker{}
			p.updateConfig(func(conf *config) { conf.maxAttachmentSize = defaultMaxAttachmentSize })

			api := &plugintest.API{}
			test.SetupAPI(api)
			p.SetAPI(api)

			defer api.AssertExpectations(t)

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "RegisterJiraCommand", func(_ *Plugin, _, _ bool) error { return nil })
			monkey.PatchInstanceMethod(reflect.TypeOf(p), "WSInstancesChanged", func(_ *Plugin, _ *Instances) {})

			jiraURL, newInstance, err := p.installCloudOAuthInstance(test.JiraURL)

			require.Nil(t, err)
			require.Equal(t, test.JiraURL, jiraURL)
			require.NotNil(t, newInstance)

			if test.HasPreviouslyInstalledInstance {
				require.Equal(t, test.JiraURL, newInstance.JWTInstance.AtlassianSecurityContext.BaseURL)
				require.NotNil(t, newInstance.JWTInstance.InstanceCommon)
				require.NotNil(t, newInstance.JWTInstance.AtlassianSecurityContext)
				require.NotEmpty(t, newInstance.JWTInstance.RawAtlassianSecurityContext)
				require.NotEmpty(t, newInstance.JWTInstance.getConfig().maxAttachmentSize)
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	for name, test := range map[string]struct {
		JiraURL                        string
		SetupAPI                       func(*plugintest.API)
		InitializeKVStore              func()
		HasPreviouslyInstalledInstance bool
	}{
		"GetClient: successfully returned a JWT client if oauth token is nil and JWT instance is stored inside oauth instance": {
			JiraURL:                        "https://brightscout-dev.atlassian.net",
			HasPreviouslyInstalledInstance: true,
			SetupAPI: func(api *plugintest.API) {
				api.On("LogDebug", "Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil").Once()
				SetData(cloudOAuthInstanceDataWithJWTInstance, api)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := &Plugin{}
			p.instanceStore = mockInstanceStoreForOauthMigration{plugin: p}

			api := &plugintest.API{}
			test.SetupAPI(api)
			p.SetAPI(api)

			defer api.AssertExpectations(t)

			instance, err := p.instanceStore.LoadInstance(types.ID(test.JiraURL))
			require.Nil(t, err)

			jiraClient, err := instance.GetClient(&Connection{})
			require.Nil(t, err)
			require.NotNil(t, jiraClient)
		})
	}
}
