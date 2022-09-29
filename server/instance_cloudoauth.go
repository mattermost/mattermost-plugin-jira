package main

import (
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/oauther"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type cloudOAuthInstance struct {
	*InstanceCommon

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	// TODO Not sure if this is also necessary for OAuth or only for Atlassian Connect
	*AtlassianSecurityContext `json:"-"`
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
	ci.Plugin.OAuther = oauther.NewFromClient(
		ci.Plugin.client,
		ci.getOAuthConfig(),
		ci.onConnect,
		logger.New(ci.Plugin.API),
	)

	// TODO I think this part is wrong, review the entire flow
	_, err := ci.Plugin.OAuther.GetToken(connection.AccountID) // TODO I don't think this AccountID is the right one
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}

	jiraClient, err := jira.NewClient(httpClient, ci.GetURL())
	if err != nil {
		return nil, err
	}

	return newCloudClient(jiraClient), nil
}

func (ci *cloudOAuthInstance) getOAuthConfig() oauth2.Config {
	return oauth2.Config{
		ClientID:     ci.AtlassianSecurityContext.OAuthClientID,
		ClientSecret: ci.AtlassianSecurityContext.SharedSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.io",
			TokenURL: "https://auth.atlassian.io/oauth2/token",
		},
	}
}

func (ci *cloudOAuthInstance) onConnect(userID string, token oauth2.Token, payload []byte) {
	// TODO
}

func (ci *cloudOAuthInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Jira Cloud Mattermost Key": ci.MattermostKey,
	}
}

func (ci *cloudOAuthInstance) GetUserConnectURL(mattermostUserID string) (string, *http.Cookie, error) {
	// TODO
	ci.Plugin.OAuther = oauther.NewFromClient(
		ci.Plugin.client,
		ci.getOAuthConfig(),
		ci.onConnect,
		logger.New(ci.Plugin.API),
	)

	return "", &http.Cookie{}, nil
}

func (ci *cloudOAuthInstance) GetURL() string {
	return ci.InstanceID.String()
}

func (ci *cloudOAuthInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/plugins/servlet/applinks/listApplicationLinks", ci.GetURL())
}

func (ci *cloudOAuthInstance) GetManageWebhooksURL() string {
	return fmt.Sprintf("%s/plugins/servlet/webhooks", ci.GetURL())
}
