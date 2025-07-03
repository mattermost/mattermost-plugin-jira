// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.enterprise for license information.

package enterprise

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Checker interface {
	HasEnterpriseFeatures() bool
}

type enterpriseChecker struct {
	api PluginAPI
}

type PluginAPI interface {
	GetLicense() *model.License
	GetConfig() *model.Config
}

func NewEnterpriseChecker(api PluginAPI) Checker {
	return &enterpriseChecker{
		api: api,
	}
}

func (e *enterpriseChecker) HasEnterpriseFeatures() bool {
	config := e.api.GetConfig()
	license := e.api.GetLicense()

	return pluginapi.IsE10LicensedOrDevelopment(config, license)
}
