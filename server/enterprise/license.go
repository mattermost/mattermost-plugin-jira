package enterprise

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

type EnterpriseChecker interface {
	HasEnterpriseFeatures() bool
}

type enterpriseChecker struct {
	api PluginAPI
}

type PluginAPI interface {
	GetLicense() *model.License
}

func NewEnterpriseChecker(api PluginAPI) EnterpriseChecker {
	return &enterpriseChecker{
		api: api,
	}
}

func (e *enterpriseChecker) HasEnterpriseFeatures() bool {
	license := e.api.GetLicense()
	if license == nil {
		return false
	}

	if license.Features == nil {
		return false
	}

	if license.Features.EnterprisePlugins == nil {
		return false
	}

	if !*license.Features.EnterprisePlugins {
		return false
	}

	return true
}
