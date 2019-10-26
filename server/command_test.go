package main

import (
	"errors"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

const (
	mockUserIDWithNotifications    = "1"
	mockUserIDWithoutNotifications = "2"
	mockUserIDUnknown              = "3"
)

type mockUserStoreKV struct {
	mockUserStore
	kv map[string]JIRAUser
}

func (store mockUserStoreKV) LoadJIRAUser(ji Instance, mattermostUserId string) (JIRAUser, error) {
	user, ok := store.kv[mattermostUserId]
	if !ok {
		return JIRAUser{}, errors.New("user not found")
	}
	return user, nil
}

func getMockUserStoreKV() mockUserStoreKV {
	return mockUserStoreKV{
		kv: map[string]JIRAUser{
			mockUserIDWithNotifications:    {Settings: &UserSettings{Notifications: true}},
			mockUserIDWithoutNotifications: {Settings: &UserSettings{Notifications: false}},
		},
	}
}

func TestPlugin_ExecuteCommand_Settings(t *testing.T) {
	p := Plugin{}
	tc := TestConfiguration{}
	p.updateConfig(func(conf *config) {
		conf.Secret = tc.Secret
	})
	api := &plugintest.API{}
	siteURL := "https://somelink.com"
	api.On("GetConfig").Return(&model.Config{ServiceSettings: model.ServiceSettings{SiteURL: &siteURL}})
	api.On("LogError", mock.AnythingOfTypeArgument("string")).Return(nil)

	tests := map[string]struct {
		commandArgs                *model.CommandArgs
		initializeEmptyUserStorage bool
		expectedMsg                string
	}{
		"no storage": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings", UserId: mockUserIDUnknown},
			initializeEmptyUserStorage: true,
			expectedMsg:                "Failed to load current Jira instance. Please contact your system administrator.",
		},
		"user not found": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings", UserId: mockUserIDUnknown},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Your username is not connected to Jira. Please type `jira connect`. user not found",
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
			commandArgs:                &model.CommandArgs{Command: "/jira settings test", UserId: mockUserIDWithoutNotifications},
			initializeEmptyUserStorage: false,
			expectedMsg:                "Unknown setting.",
		},
		"set notifications without value": {
			commandArgs:                &model.CommandArgs{Command: "/jira settings notifications", UserId: mockUserIDWithoutNotifications},
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
			if tt.initializeEmptyUserStorage {
				p.currentInstanceStore = mockCurrentInstanceStoreNoInstance{}
			} else {
				p.currentInstanceStore = mockCurrentInstanceStore{}
			}
			p.userStore = getMockUserStoreKV()

			p.ExecuteCommand(&plugin.Context{}, tt.commandArgs)

			assert.Equal(t, true, isSendEphemeralPostCalled)
		})
	}
}
