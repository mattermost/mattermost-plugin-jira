package main

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-jira/server/enterprise"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInstallInstance(t *testing.T) {
	trueValue := true
	p := &Plugin{}

	for name, tc := range map[string]struct {
		license      *model.License
		numInstances int
		expectError  bool
	}{
		"0 preinstalled   valid license": {
			numInstances: 0,
			expectError:  false,
			license: &model.License{
				Features: &model.Features{
					EnterprisePlugins: &trueValue,
				},
			},
		},
		"0 preinstalled   nil license": {
			numInstances: 0,
			expectError:  false,
			license:      nil,
		},
		"0 preinstalled   nil Features": {
			numInstances: 0,
			expectError:  false,
			license:      &model.License{},
		},
		"0 preinstalled   nil Features EnterprisePlugins": {
			numInstances: 0,
			expectError:  false,
			license: &model.License{
				Features: &model.Features{},
			},
		},
		"1 preinstalled   valid license": {
			numInstances: 1,
			expectError:  false,
			license: &model.License{
				Features: &model.Features{
					EnterprisePlugins: &trueValue,
				},
			},
		},
		"1 preinstalled   nil license": {
			numInstances: 1,
			expectError:  true,
			license:      nil,
		},
		"1 preinstalled   nil Features": {
			numInstances: 1,
			expectError:  true,
			license:      &model.License{},
		},
		"1 preinstalled   nil Features EnterprisePlugins": {
			numInstances: 1,
			expectError:  true,
			license: &model.License{
				Features: &model.Features{},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			p.SetAPI(api)
			p.enterpriseChecker = enterprise.NewEnterpriseChecker(api)
			p.instanceStore = p.getMockInstanceStoreKV(tc.numInstances)

			api.On("KVGet", mock.Anything).Return(mock.Anything, nil)
			api.On("GetLicense").Return(tc.license)
			api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil)
			api.On("RegisterCommand", mock.Anything, mock.Anything).Return(nil)
			api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything)

			testInstance0 := &testInstance{
				InstanceCommon: InstanceCommon{
					InstanceID: mockInstance1URL,
					IsV2Legacy: true,
					Type:       "testInstanceType",
				},
			}

			err := p.InstallInstance(testInstance0)
			if tc.expectError {
				assert.NotNil(t, err)
				expected := "You need an Enterprise License to install multiple Jira instances"
				assert.Equal(t, expected, err.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
