package main

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"sync"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	mockUserIDWithNotifications    = "1"
	mockUserIDWithoutNotifications = "2"
	mockUserIDUnknown              = "3"
	mockUserIDSysAdmin             = "4"
	mockUserIDNonSysAdmin          = "5"
)

type mockUserStoreKV struct {
	mockUserStore
	connections map[types.ID]*Connection
	users       map[types.ID]*User
}

var _ UserStore = (*mockUserStoreKV)(nil)

func (store mockUserStoreKV) LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error) {
	connection, ok := store.connections[mattermostUserID]
	if !ok {
		return nil, errors.Errorf("TESTING connection %q %q not found", instanceID, mattermostUserID)
	}
	return connection, nil
}

func (store mockUserStoreKV) LoadUser(mattermostUserID types.ID) (*User, error) {
	user, ok := store.users[mattermostUserID]
	if !ok {
		return nil, errors.Errorf("TESTING user %q not found", mattermostUserID)
	}
	return user, nil
}

func getMockUserStoreKV() mockUserStoreKV {
	newuser := func(id types.ID) *User {
		u := NewUser(id)
		u.ConnectedInstances.Set(testInstance1.Common())
		return u
	}

	connection := Connection{
		User: jira.User{
			AccountID: "test",
		},
	}

	withNotifications := connection // copy
	withNotifications.Settings = &ConnectionSettings{Notifications: true}

	return mockUserStoreKV{
		users: map[types.ID]*User{
			"connected_user":               newuser("connected_user"),
			mockUserIDWithNotifications:    newuser(mockUserIDWithNotifications),
			mockUserIDWithoutNotifications: newuser(mockUserIDWithoutNotifications),
		},
		connections: map[types.ID]*Connection{
			mockUserIDWithNotifications:    &withNotifications,
			mockUserIDWithoutNotifications: &connection,
			"connected_user":               &connection,
		},
	}
}

type mockInstanceStoreKV struct {
	mockInstanceStore
	kv *sync.Map
	*Instances
	*Plugin
}

var _ InstanceStore = (*mockInstanceStoreKV)(nil)

func (store *mockInstanceStoreKV) LoadInstances() (*Instances, error) {
	return store.Instances, nil
}

func (store *mockInstanceStoreKV) LoadInstance(id types.ID) (Instance, error) {
	v, ok := store.kv.Load(id)
	if !ok {
		return nil, errors.Errorf("instance %q not found", id)
	}
	instance := v.(Instance)
	return instance, nil
}

func (p *Plugin) getMockInstanceStoreKV(uninitialized bool) *mockInstanceStoreKV {
	kv := sync.Map{}
	instances := NewInstances()
	if !uninitialized {
		for _, ti := range []*testInstance{testInstance1, testInstance2} {
			instance := *ti
			instance.Plugin = p
			instances.Set(instance.Common())
			kv.Store(instance.GetID(), &instance)
		}
	}
	return &mockInstanceStoreKV{
		kv:        &kv,
		Instances: instances,
		Plugin:    p,
	}
}

func TestPlugin_ExecuteCommand_Settings(t *testing.T) {
	p := &Plugin{}
	tc := TestConfiguration{}
	p.updateConfig(func(conf *config) {
		conf.Secret = tc.Secret
		conf.mattermostSiteURL = "https://somelink.com"
	})
	api := &plugintest.API{}
	api.On("LogError", mock.AnythingOfTypeArgument("string")).Return(nil)

	tests := map[string]struct {
		commandArgs                *model.CommandArgs
		initializeEmptyUserStorage bool
		expectedMsg                string
	}{
		"no storage": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings", UserId: mockUserIDUnknown},
			initializeEmptyUserStorage: true,
			expectedMsg:                "Failed to load your connection to Jira. Error: TESTING user \"3\" not found.",
		},
		"user not found": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings", UserId: mockUserIDUnknown},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Failed to load your connection to Jira. Error: TESTING user \"3\" not found.",
		},
		"no params, with notifications": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings", UserId: mockUserIDWithNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Current settings:\n\tNotifications: on",
		},
		"no params, without notifications": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings", UserId: mockUserIDWithoutNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Current settings:\n\tNotifications: off",
		},
		"unknown setting": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings" + " test", UserId: mockUserIDWithoutNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Unknown setting.",
		},
		"set notifications without value": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings" + " notifications", UserId: mockUserIDWithoutNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "`/jira settings notifications [value]`\n* Invalid value. Accepted values are: `on` or `off`.",
		},
		"set notification with unknown value": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings notifications test", UserId: mockUserIDWithoutNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "`/jira settings notifications [value]`\n* Invalid value. Accepted values are: `on` or `off`.",
		},
		"enable notifications": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings notifications on", UserId: mockUserIDWithoutNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Settings updated. Notifications on.",
		},
		"disable notifications": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings notifications off", UserId: mockUserIDWithNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Settings updated. Notifications off.",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			isSendEphemeralPostCalled := false

			currentTestApi := api
			currentTestApi.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Run(func(args mock.Arguments) {
				isSendEphemeralPostCalled = true

				post := args.Get(1).(*model.Post)
				assert.Equal(t, tt.expectedMsg, post.Message)
			}).Once().Return(&model.Post{})

			p.SetAPI(currentTestApi)
			p.instanceStore = p.getMockInstanceStoreKV(tt.initializeEmptyUserStorage)
			p.userStore = getMockUserStoreKV()

			p.ExecuteCommand(&plugin.Context{}, tt.commandArgs)

			assert.Equal(t, true, isSendEphemeralPostCalled)
		})
	}
}

func TestPlugin_ExecuteCommand_Installation(t *testing.T) {
	p := Plugin{}
	tc := TestConfiguration{}
	p.updateConfig(func(conf *config) {
		conf.Secret = tc.Secret
		conf.mattermostSiteURL = "https://somelink.com"
		conf.rsaKey, _ = rsa.GenerateKey(rand.Reader, 1024)
	})
	api := &plugintest.API{}
	api.On("LogError", mock.AnythingOfTypeArgument("string")).Return(nil)
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
	api.On("KVSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
	api.On("KVSetWithExpiry", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
	api.On("KVGet", keyInstances).Return(nil, nil)
	api.On("KVGet", "rsa_key").Return(nil, nil)
	api.On("PublishWebSocketEvent", mock.AnythingOfTypeArgument("string"), mock.Anything, mock.Anything)
	// api.On("GetTeam", mock.AnythingOfTypeArgument("string")).Return(&model.Team{Name: "TestTeam"}, nil)
	// api.On("GetChannel", mock.AnythingOfTypeArgument("string")).Return(&model.Channel{Name: "TestChannel"}, nil)
	api.On("UnregisterCommand", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	sysAdminUser := &model.User{
		Id:    mockUserIDSysAdmin,
		Roles: "system_admin",
	}
	api.On("GetUser", mockUserIDSysAdmin).Return(sysAdminUser, nil)
	nonSysAdminUser := &model.User{
		Id:    mockUserIDNonSysAdmin,
		Roles: "",
	}
	api.On("GetUser", mockUserIDNonSysAdmin).Return(nonSysAdminUser, nil)

	tests := map[string]struct {
		commandArgs       *model.CommandArgs
		expectedMsgPrefix string
	}{
		"no params - user is sys admin": {
			commandArgs:       &model.CommandArgs{Command: "/jira install", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText + sysAdminHelpText),
		},
		"no params - user is not sys admin": {
			commandArgs:       &model.CommandArgs{Command: "/jira install", UserId: mockUserIDNonSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText),
		},
		"install server without URL": {
			commandArgs:       &model.CommandArgs{Command: "/jira install server", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText + sysAdminHelpText),
		},
		"install cloud instance without URL": {
			commandArgs:       &model.CommandArgs{Command: "/jira install cloud", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText + sysAdminHelpText),
		},
		"install cloud instance as server": {
			commandArgs:       &model.CommandArgs{Command: "/jira install server https://mmtest.atlassian.net", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: "The Jira URL you provided looks like a Jira Cloud URL",
		},
		"install server instance using mattermost site URL": {
			commandArgs:       &model.CommandArgs{Command: "/jira install server https://somelink.com", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: "https://somelink.com is the Mattermost site URL. Please use your Jira URL with `/jira install`.",
		},
		"install valid cloud instance": {
			commandArgs:       &model.CommandArgs{Command: "/jira install cloud https://mmtest.atlassian.net", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: "https://mmtest.atlassian.net has been successfully installed.",
		},
		"install non secure cloud instance": {
			commandArgs:       &model.CommandArgs{Command: "/jira install cloud http://mmtest.atlassian.net", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: "`/jira install cloud` requires a secure connection (HTTPS). Please run the following command:",
		},
		"install valid server instance": {
			commandArgs:       &model.CommandArgs{Command: "/jira install server https://jiralink.com", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: "Server instance has been installed",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			isSendEphemeralPostCalled := false

			currentTestApi := api
			currentTestApi.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Run(func(args mock.Arguments) {
				isSendEphemeralPostCalled = true

				post := args.Get(1).(*model.Post)
				actual := strings.TrimSpace(post.Message)
				assert.True(t, strings.HasPrefix(actual, tt.expectedMsgPrefix), "Expected returned message to start with: \n%s\nActual:\n%s", tt.expectedMsgPrefix, actual)
			}).Once().Return(&model.Post{})

			p.SetAPI(currentTestApi)

			store := NewStore(&p)
			p.instanceStore = store
			p.secretsStore = store
			p.userStore = getMockUserStoreKV()

			cmdResponse, appError := p.ExecuteCommand(&plugin.Context{}, tt.commandArgs)
			require.Nil(t, appError)
			require.NotNil(t, cmdResponse)
			assert.True(t, isSendEphemeralPostCalled)
		})
	}
}

func TestPlugin_ExecuteCommand_Uninstall(t *testing.T) {
	p := Plugin{}
	tc := TestConfiguration{}
	p.updateConfig(func(conf *config) {
		conf.Secret = tc.Secret
		conf.mattermostSiteURL = "https://somelink.com"
	})
	api := &plugintest.API{}

	sysAdminUser := &model.User{
		Id:    mockUserIDSysAdmin,
		Roles: "system_admin",
	}
	api.On("GetUser", mockUserIDSysAdmin).Return(sysAdminUser, nil)
	nonSysAdminUser := &model.User{
		Id:    mockUserIDNonSysAdmin,
		Roles: "",
	}
	api.On("GetUser", mockUserIDNonSysAdmin).Return(nonSysAdminUser, nil)

	tests := map[string]struct {
		commandArgs       *model.CommandArgs
		expectedMsgPrefix string
	}{
		"no params - user is sys admin": {
			commandArgs:       &model.CommandArgs{Command: "/jira uninstall", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText + sysAdminHelpText),
		},
		"no params - user is not sys admin": {
			commandArgs:       &model.CommandArgs{Command: "/jira uninstall", UserId: mockUserIDNonSysAdmin},
			expectedMsgPrefix: "`/jira uninstall` can only be run by a System Administrator.",
		},
		"uninstall with invalid option": {
			commandArgs:       &model.CommandArgs{Command: "/jira uninstall foo", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText + sysAdminHelpText),
		},
		"uninstall server instance without URL": {
			commandArgs:       &model.CommandArgs{Command: "/jira uninstall server", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText + sysAdminHelpText),
		},
		"uninstall cloud instance without URL": {
			commandArgs:       &model.CommandArgs{Command: "/jira uninstall cloud", UserId: mockUserIDSysAdmin},
			expectedMsgPrefix: strings.TrimSpace(helpTextHeader + commonHelpText + sysAdminHelpText),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			isSendEphemeralPostCalled := false
			currentTestAPI := api
			currentTestAPI.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Run(func(args mock.Arguments) {
				isSendEphemeralPostCalled = true

				post := args.Get(1).(*model.Post)
				actual := strings.TrimSpace(post.Message)
				assert.True(t, strings.HasPrefix(actual, tt.expectedMsgPrefix), "Expected returned message to start with: \n%s\nActual:\n%s", tt.expectedMsgPrefix, actual)
			}).Once().Return(&model.Post{})

			p.SetAPI(currentTestAPI)

			cmdResponse, appError := p.ExecuteCommand(&plugin.Context{}, tt.commandArgs)
			require.Nil(t, appError)
			require.NotNil(t, cmdResponse)
			assert.True(t, isSendEphemeralPostCalled)
		})
	}
}
