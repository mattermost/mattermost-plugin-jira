package enterprise

import (
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v5/model"
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
