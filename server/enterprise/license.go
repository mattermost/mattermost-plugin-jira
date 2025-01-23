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

const (
	e20          = "E20"
	professional = "professional"
	enterprise   = "enterprise"
)

func (e *enterpriseChecker) HasEnterpriseFeatures() bool {
	config := e.api.GetConfig()
	license := e.api.GetLicense()

	if license != nil && (license.SkuShortName == e20 || license.SkuShortName == enterprise || license.SkuShortName == professional) {
		return true
	}

	return pluginapi.IsE20LicensedOrDevelopment(config, license)
}
