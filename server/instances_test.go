// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"path/filepath"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/enterprise"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func TestInstallInstance(t *testing.T) {
	trueValue := true
	p := &Plugin{}

	for name, tc := range map[string]struct {
		license      *model.License
		numInstances int
		expectError  bool
		devEnabled   bool
	}{
		"0 preinstalled, valid license": {
			numInstances: 0,
			expectError:  false,
			license: &model.License{
				SkuShortName: "professional",
			},
		},
		"0 preinstalled, nil license": {
			numInstances: 0,
			expectError:  false,
			license:      nil,
		},
		"1 preinstalled, professional license": {
			numInstances: 1,
			expectError:  false,
			license: &model.License{
				SkuShortName: "professional",
			},
		},
		"1 preinstalled, Enterprise Advanced license": {
			numInstances: 1,
			expectError:  false,
			license: &model.License{
				SkuShortName: "advanced",
			},
		},
		"1 preinstalled, enterprise license": {
			numInstances: 1,
			expectError:  false,
			license: &model.License{
				SkuShortName: "enterprise",
			},
		},
		"1 preinstalled, cloud starter license. should have error": {
			numInstances: 1,
			expectError:  true,
			license: &model.License{
				SkuShortName: "starter",
			},
		},
		"1 preinstalled, dev mode": {
			numInstances: 1,
			expectError:  false,
			license:      nil,
			devEnabled:   true,
		},
		"1 preinstalled  nil license": {
			numInstances: 1,
			expectError:  true,
			license:      nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)
			p.enterpriseChecker = enterprise.NewEnterpriseChecker(api)
			p.instanceStore = p.getMockInstanceStoreKV(tc.numInstances)

			conf := &model.Config{}
			if tc.devEnabled {
				conf.ServiceSettings.EnableDeveloper = &trueValue
				conf.ServiceSettings.EnableTesting = &trueValue
			}

			api.On("KVGet", mock.Anything).Return(mock.Anything, nil)
			api.On("GetLicense").Return(tc.license)
			api.On("GetConfig").Return(conf)
			api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil)
			api.On("RegisterCommand", mock.Anything, mock.Anything).Return(nil)
			api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything)

			path, err := filepath.Abs("..")
			require.Nil(t, err)
			api.On("GetBundlePath").Return(path, nil)

			testInstance0 := &testInstance{
				InstanceCommon: InstanceCommon{
					InstanceID: mockInstance3URL,
					IsV2Legacy: true,
					Type:       "testInstanceType",
				},
			}

			err = p.InstallInstance(testInstance0)
			if tc.expectError {
				assert.NotNil(t, err)
				expected := "You need a valid Mattermost Professional, Enterprise or Enterprise Advanced License to install multiple Jira instances."
				assert.Equal(t, expected, err.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestResolveUserInstanceURL_StaleInstances covers a user record that still
// references an uninstalled instance. Such a reference used to be returned
// as the resolved instance, so every instance-resolving command failed on
// it, including the disconnect that would have cleaned it up.
func TestResolveUserInstanceURL_StaleInstances(t *testing.T) {
	deadInstanceID := types.ID("https://dead-instance.example.com")
	p := newPluginForStoreTests(t, newInstanceStoreDouble(testInstance1))

	t.Run("a default pointing at an uninstalled instance falls back to the installed one", func(t *testing.T) {
		user := NewUser("test-user")
		user.ConnectedInstances.Set(testInstance1.Common())
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
		user.DefaultInstanceID = deadInstanceID

		instanceID, err := p.resolveUserInstanceURL(user, "")
		require.NoError(t, err)
		assert.Equal(t, testInstance1.InstanceID, instanceID)
	})

	t.Run("being connected only to an uninstalled instance reports not-connected", func(t *testing.T) {
		user := NewUser("test-user")
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
		user.DefaultInstanceID = deadInstanceID

		_, err := p.resolveUserInstanceURL(user, "")
		require.Error(t, err)
		assert.Equal(t, kvstore.ErrNotFound, errors.Cause(err))
	})

	t.Run("an explicitly named instance is honored even when uninstalled", func(t *testing.T) {
		user := NewUser("test-user")
		user.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})

		instanceID, err := p.resolveUserInstanceURL(user, deadInstanceID.String())
		require.NoError(t, err)
		assert.Equal(t, deadInstanceID, instanceID, "/jira disconnect has to be able to target a removed instance")
	})
}

func TestUninstallInstance(t *testing.T) {
	t.Run("the instance list is written before the instance blob is deleted", func(t *testing.T) {
		store := newInstanceStoreDouble(testInstance1)
		p := newPluginForStoreTests(t, store)
		p.userStore = mockUserStoreKV{users: map[types.ID]*User{}, connections: map[types.ID]*Connection{}}

		_, _, err := p.UninstallInstance(testInstance1.InstanceID, testInstance1.Type)
		require.NoError(t, err)
		assert.Equal(t, []string{"StoreInstances", "DeleteInstance"}, store.writes,
			"deleting the blob first leaves the list pointing at a dead instance if the list write then fails")
	})

	t.Run("a missing instance blob still removes the list entry and disconnects users", func(t *testing.T) {
		deadInstanceID := types.ID("https://dead-instance.example.com")

		// An install already in the broken state: listed in instances/v3,
		// but with no blob behind it.
		store := newInstanceStoreDouble()
		store.instances.Set(&InstanceCommon{InstanceID: deadInstanceID, Type: ServerInstanceType})

		affected := NewUser("affected-user")
		affected.ConnectedInstances.Set(&InstanceCommon{InstanceID: deadInstanceID})
		affected.DefaultInstanceID = deadInstanceID
		unaffected := NewUser("unaffected-user")
		unaffected.ConnectedInstances.Set(testInstance1.Common())

		p := newPluginForStoreTests(t, store)
		p.userStore = mockUserStoreKV{
			users: map[types.ID]*User{
				affected.MattermostUserID:   affected,
				unaffected.MattermostUserID: unaffected,
			},
			connections: map[types.ID]*Connection{
				affected.MattermostUserID: {User: jira.User{AccountID: "dead-account"}},
			},
		}

		instance, failedUsers, err := p.UninstallInstance(deadInstanceID, ServerInstanceType)
		require.NoError(t, err, "must not error, and must not panic, on a missing instance blob")
		require.NotNil(t, instance, "callers print manage-app URLs from the returned instance")
		assert.Equal(t, 0, failedUsers)

		instances, err := p.instanceStore.LoadInstances()
		require.NoError(t, err)
		assert.False(t, instances.Contains(deadInstanceID))

		updated, err := p.userStore.LoadUser(affected.MattermostUserID)
		require.NoError(t, err)
		assert.False(t, updated.ConnectedInstances.Contains(deadInstanceID))
		assert.Empty(t, updated.DefaultInstanceID)

		untouched, err := p.userStore.LoadUser(unaffected.MattermostUserID)
		require.NoError(t, err)
		assert.True(t, untouched.ConnectedInstances.Contains(testInstance1.InstanceID))
	})
}
