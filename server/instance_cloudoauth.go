package main

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/pkg/errors"
)

type cloudOAuthInstance struct {
	*InstanceCommon

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string
}

func (p *Plugin) installCloudOAuthInstance(rawURL string) (string, *cloudOAuthInstance, error) {
	jiraURL, err := utils.CheckJiraURL(p.GetSiteURL(), rawURL, false)
	if err != nil {
		return "", nil, err
	}
	if !utils.IsJiraCloudURL(jiraURL) {
		return "", nil, errors.Errorf("`%s` is a Jira server URL, not a Jira Cloud", jiraURL)
	}

	instance := &cloudOAuthInstance{
		InstanceCommon: newInstanceCommon(p, ServerInstanceType, types.ID(jiraURL)),
		MattermostKey:  p.GetPluginKey(),
	}

	err = p.InstallInstance(instance)
	if err != nil {
		return "", nil, err
	}

	return jiraURL, instance, err
}

func (ci *cloudOAuthInstance) GetClient(connection *Connection) (Client, error) {
	// TODO See instance_server.GetClient
}

func (ci *cloudOAuthInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Atlassian Connect Key":        "", // TODO
		"Atlassian Connect Client Key": "",
		"Jira Cloud Version":           "",
		"Jira Cloud Plugins Version":   "",
	}
}

func (ci *cloudOAuthInstance) GetUserConnectURL(mattermostUserID string) (string, *http.Cookie, error) {
	// TODO
}

func (ci *cloudOAuthInstance) GetURL() string {
	return "" // TODO BaseURL
}

func (ci *cloudOAuthInstance) GetManageAppsURL() string {
	// TODO
}

func (ci *cloudOAuthInstance) GetManageWebhooksURL() string {
	// TODO
}
