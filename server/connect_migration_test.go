// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	connectTestBotUserID  = "test-bot-user-id"
	connectTestChannelID  = "test-channel-id"
	connectTestAdminID    = "test-admin-id"
	connectTestJiraURL    = "https://mmtest.atlassian.net"
	connectTestInstanceID = types.ID(connectTestJiraURL)
)

type mapUsersStore struct {
	*mockUserStoreForTokenExpiry
	users []*User
}

func (s *mapUsersStore) MapUsers(f func(*User) error) error {
	for _, user := range s.users {
		if err := f(user); err != nil {
			return err
		}
	}
	return nil
}

func newConnectTestPlugin(api *plugintest.API, userStore UserStore, instanceStore InstanceStore) *Plugin {
	p := &Plugin{
		userStore:     userStore,
		instanceStore: instanceStore,
		tracker:       &mockTelemetryTracker{},
	}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.updateConfig(func(conf *config) {
		conf.botUserID = connectTestBotUserID
	})
	return p
}

func expectAdminDM(api *plugintest.API, message string) {
	api.On("GetUsers", mock.MatchedBy(func(o *model.UserGetOptions) bool {
		return o.Role == model.SystemAdminRoleId && o.Active
	})).Return([]*model.User{{Id: connectTestAdminID}, {Id: "admin-bot", IsBot: true}}, nil).Once()
	api.On("GetDirectChannel", connectTestAdminID, connectTestBotUserID).Return(&model.Channel{Id: connectTestChannelID}, nil).Once()
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == connectTestChannelID && post.Message == message
	})).Return(&model.Post{}, nil).Once()
}

func legacyConnectInstance() *cloudInstance {
	return &cloudInstance{
		InstanceCommon:           &InstanceCommon{InstanceID: connectTestInstanceID, Type: CloudInstanceType},
		AtlassianSecurityContext: &AtlassianSecurityContext{BaseURL: connectTestJiraURL},
	}
}

func TestLegacyConnectInstanceIsRefused(t *testing.T) {
	ci := legacyConnectInstance()

	_, err := ci.GetClient(&Connection{MattermostUserID: "someuser"})
	require.ErrorIs(t, err, errConnectInstanceUnsupported)

	_, _, err = ci.GetUserConnectURL("someuser")
	require.ErrorIs(t, err, errConnectInstanceUnsupported)
}

func TestCloudOAuthClientForConnectConnection(t *testing.T) {
	mmUserID := types.ID("test-mm-user-id")

	t.Run("Connect-era connection is disconnected and asked to reconnect", func(t *testing.T) {
		api := &plugintest.API{}
		userStore := &mockUserStoreForTokenExpiry{}
		instanceStore := &mockInstanceStoreWithLoadInstances{mockInstanceStore: &mockInstanceStore{}}
		p := newConnectTestPlugin(api, userStore, instanceStore)

		user := NewUser(mmUserID)
		user.ConnectedInstances = NewInstances()
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: connectTestInstanceID})
		userStore.On("LoadUser", mmUserID).Return(user, nil).Once()
		userStore.On("LoadConnection", connectTestInstanceID, mmUserID).Return(&Connection{MattermostUserID: mmUserID}, nil).Once()
		userStore.On("DeleteConnection", connectTestInstanceID, mmUserID).Return(nil).Once()
		userStore.On("StoreUser", mock.AnythingOfType("*main.User")).Return(nil).Once()

		instanceStore.On("LoadInstance", connectTestInstanceID).Return(&testInstance{InstanceCommon: InstanceCommon{InstanceID: connectTestInstanceID}}, nil).Once()
		instances := NewInstances()
		instances.Set(&InstanceCommon{InstanceID: connectTestInstanceID})
		instanceStore.On("LoadInstances").Return(instances, nil).Twice()

		api.On("KVSetWithOptions", mock.AnythingOfType("string"), []byte("1"), mock.Anything).Return(true, (*model.AppError)(nil)).Once()
		api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)
		api.On("PublishWebSocketEvent", "disconnect", mock.Anything, mock.Anything).Return().Once()
		api.On("GetDirectChannel", mmUserID.String(), connectTestBotUserID).Return(&model.Channel{Id: connectTestChannelID}, nil)
		api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.Message == ":warning: Your Jira connection was made with the Atlassian Connect app, which is no longer supported. "+
				"Please reconnect your account using `/jira connect https://mmtest.atlassian.net`."
		})).Return(&model.Post{}, nil).Once()

		ci := &cloudOAuthInstance{InstanceCommon: &InstanceCommon{Plugin: p, InstanceID: connectTestInstanceID, Type: CloudOAuthInstanceType}}
		_, err := ci.GetClient(&Connection{MattermostUserID: mmUserID})
		require.ErrorContains(t, err, "/jira connect")

		api.AssertExpectations(t)
		userStore.AssertExpectations(t)
		instanceStore.AssertExpectations(t)
	})

	t.Run("user who never connected gets no message", func(t *testing.T) {
		api := &plugintest.API{}
		p := newConnectTestPlugin(api, &mockUserStoreForTokenExpiry{}, &mockInstanceStore{})

		ci := &cloudOAuthInstance{InstanceCommon: &InstanceCommon{Plugin: p, InstanceID: connectTestInstanceID, Type: CloudOAuthInstanceType}}
		_, err := ci.GetClient(&Connection{})
		require.ErrorContains(t, err, "not connected")

		api.AssertExpectations(t)
	})
}

func TestMigrateAwayFromConnect(t *testing.T) {
	t.Run("admins are asked once to reinstall a Connect instance", func(t *testing.T) {
		api := &plugintest.API{}
		instanceStore := &mockInstanceStore{}
		p := newConnectTestPlugin(api, &mockUserStoreForTokenExpiry{}, instanceStore)
		p.updateConfig(func(conf *config) {
			conf.AdminAPIToken = "token"
			conf.AdminEmail = "admin@example.com"
		})

		instanceStore.On("LoadInstance", connectTestInstanceID).Return(legacyConnectInstance(), nil).Twice()
		api.On("KVSetWithOptions", mock.AnythingOfType("string"), []byte("1"), mock.Anything).Return(true, (*model.AppError)(nil)).Once()
		api.On("KVSetWithOptions", mock.AnythingOfType("string"), []byte("1"), mock.Anything).Return(false, (*model.AppError)(nil)).Once()
		expectAdminDM(api, fmt.Sprintf(legacyConnectInstanceNotice, connectTestJiraURL, connectTestJiraURL+"/plugins/servlet/upm"))

		instances := NewInstances()
		instances.Set(&InstanceCommon{InstanceID: connectTestInstanceID, Type: CloudInstanceType})
		p.migrateAwayFromConnect(instances)
		p.migrateAwayFromConnect(instances)

		api.AssertExpectations(t)
		instanceStore.AssertExpectations(t)
	})

	t.Run("admins are told to set the Admin API Token for an OAuth instance", func(t *testing.T) {
		api := &plugintest.API{}
		instanceStore := &mockInstanceStore{}
		p := newConnectTestPlugin(api, &mockUserStoreForTokenExpiry{}, instanceStore)

		instanceStore.On("LoadInstance", connectTestInstanceID).Return(&cloudOAuthInstance{
			InstanceCommon: &InstanceCommon{InstanceID: connectTestInstanceID, Type: CloudOAuthInstanceType},
			JiraBaseURL:    connectTestJiraURL,
		}, nil).Once()
		api.On("KVSetWithOptions", mock.AnythingOfType("string"), []byte("1"), mock.Anything).Return(true, (*model.AppError)(nil)).Twice()
		expectAdminDM(api, fmt.Sprintf(missingAPITokenNotice, connectTestJiraURL))

		instances := NewInstances()
		instances.Set(&InstanceCommon{InstanceID: connectTestInstanceID, Type: CloudOAuthInstanceType})
		p.migrateAwayFromConnect(instances)

		api.AssertExpectations(t)
		instanceStore.AssertExpectations(t)
	})
}

func TestDisconnectConnectUsers(t *testing.T) {
	connectUserID := types.ID("connect-user")
	oauthUserID := types.ID("oauth-user")
	otherUserID := types.ID("other-user")

	connectedTo := func(id types.ID, instanceIDs ...types.ID) *User {
		user := NewUser(id)
		user.ConnectedInstances = NewInstances()
		for _, instanceID := range instanceIDs {
			user.ConnectedInstances.Set(&InstanceCommon{InstanceID: instanceID})
		}
		return user
	}

	api := &plugintest.API{}
	userStore := &mapUsersStore{
		mockUserStoreForTokenExpiry: &mockUserStoreForTokenExpiry{},
		users: []*User{
			connectedTo(connectUserID, connectTestInstanceID),
			connectedTo(oauthUserID, connectTestInstanceID),
			connectedTo(otherUserID, "https://other.atlassian.net"),
		},
	}
	p := newConnectTestPlugin(api, userStore, &mockInstanceStore{})

	userStore.On("LoadConnection", connectTestInstanceID, connectUserID).Return(&Connection{MattermostUserID: connectUserID}, nil).Once()
	userStore.On("LoadConnection", connectTestInstanceID, oauthUserID).Return(&Connection{
		MattermostUserID: oauthUserID,
		OAuth2Token:      &oauth2.Token{AccessToken: "token"},
	}, nil).Once()

	userStore.On("LoadUser", connectUserID).Return(nil, errors.New("transient kv failure")).Once()
	api.On("LogWarn", "Failed to disconnect user after token expiry",
		"mattermostUserID", connectUserID, "instanceID", connectTestInstanceID, "error", mock.Anything).Return().Once()
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), []byte("1"), mock.Anything).Return(true, (*model.AppError)(nil)).Once()
	api.On("GetDirectChannel", connectUserID.String(), connectTestBotUserID).Return(&model.Channel{Id: connectTestChannelID}, nil).Once()
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.Message == ":warning: Your Jira connection was made with the Atlassian Connect app, which is no longer supported. "+
			"Please manually disconnect and reconnect your account using:\n"+
			"1. `/jira disconnect https://mmtest.atlassian.net`\n"+
			"2. `/jira connect https://mmtest.atlassian.net`"
	})).Return(&model.Post{}, nil).Once()

	p.disconnectConnectUsers(connectTestInstanceID)

	api.AssertExpectations(t)
	userStore.AssertExpectations(t)
}

func TestExpandIssueWithoutAPITokenNotifiesAdmins(t *testing.T) {
	api := &plugintest.API{}
	p := newConnectTestPlugin(api, &mockUserStoreForTokenExpiry{}, &mockInstanceStore{})

	api.On("KVSetWithOptions", mock.AnythingOfType("string"), []byte("1"), mock.Anything).Return(true, (*model.AppError)(nil)).Once()
	expectAdminDM(api, fmt.Sprintf(missingAPITokenNotice, connectTestJiraURL))

	jwh := &JiraWebhook{}
	err := jwh.expandIssueWithAPIToken(p, legacyConnectInstance(), errConnectInstanceUnsupported)
	require.ErrorIs(t, err, errConnectInstanceUnsupported)

	api.AssertExpectations(t)
}
