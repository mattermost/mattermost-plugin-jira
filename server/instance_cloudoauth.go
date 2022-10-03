package main

import (
	"context"
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/oauther"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	oauth2_jira "golang.org/x/oauth2/jira"
)

type cloudOAuthInstance struct {
	*InstanceCommon

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string
}

var _ Instance = (*cloudOAuthInstance)(nil)

func (p *Plugin) installCloudOAuthInstance(rawURL string) (string, *cloudOAuthInstance, error) {
	jiraURL, err := utils.CheckJiraURL(p.GetSiteURL(), rawURL, false)
	if err != nil {
		return "", nil, err
	}
	if !utils.IsJiraCloudURL(jiraURL) {
		return "", nil, errors.Errorf("`%s` is a Jira server URL, not a Jira Cloud", jiraURL)
	}

	instance := &cloudOAuthInstance{
		InstanceCommon: newInstanceCommon(p, CloudOAuthInstanceType, types.ID(jiraURL)),
		MattermostKey:  p.GetPluginKey(),
	}

	err = p.InstallInstance(instance)
	if err != nil {
		return "", nil, err
	}

	return jiraURL, instance, err
}

func (ci *cloudOAuthInstance) GetClient(connection *Connection) (Client, error) {
	client, _, err := ci.getClientForConnection(connection)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get Jira client for user "+connection.DisplayName)
	}
	return newCloudClient(client), nil
}

func (ci *cloudOAuthInstance) getClientForConnection(connection *Connection) (*jira.Client, *http.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: ci.GetURL(),
		Subject: connection.AccountID,
		Config: oauth2.Config{
			ClientID:     "",
			ClientSecret: "",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://auth.atlassian.io",
				TokenURL: "https://auth.atlassian.io/oauth2/token",
			},
		},
	}

	conf := ci.getConfig()
	httpClient := oauth2Conf.Client(context.Background())
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))

	jiraClient, err := jira.NewClient(httpClient, oauth2Conf.BaseURL)
	return jiraClient, httpClient, err
}

func (ci *cloudOAuthInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Jira Cloud Mattermost Key": ci.MattermostKey,
	}
}

func (ci *cloudOAuthInstance) GetUserConnectURL(mattermostUserID string) (string, *http.Cookie, error) {
	ci.Plugin.OAuther = oauther.NewFromClient(
		ci.Plugin.client,
		ci.getOAuthConfig(),
		ci.onConnect,
		logger.New(ci.Plugin.API),
	)

	return ci.Plugin.OAuther.GetConnectURL(), nil, nil
}

func (ci *cloudOAuthInstance) getOAuthConfig() oauth2.Config {
	return oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.io",
			TokenURL: "https://auth.atlassian.io/oauth2/token",
		},
	}
}

func (ci *cloudOAuthInstance) onConnect(userID string, token oauth2.Token, payload []byte) {
	tokenKey := fmt.Sprintf("oauthcloud_token_%s", userID)
	ci.Plugin.client.KV.Set(tokenKey, token.AccessToken)
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

func (ci *cloudOAuthInstance) GetMattermostKey() string {
	return ci.MattermostKey
}
