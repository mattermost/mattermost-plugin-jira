package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
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

type OAuth2Token struct {
	AccessToken string `json:"access_token"`
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
	conf := ci.getConfig()
	oauth2Conf := oauth2_jira.Config{
		BaseURL: ci.GetURL(),
		Subject: connection.AccountID,
		Config: oauth2.Config{
			ClientID:     conf.JiraAuthAppClientID,
			ClientSecret: conf.JiraAuthAppClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://auth.atlassian.com",
				TokenURL: "https://auth.atlassian.com/oauth/token",
			},
		},
	}

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
	conf := ci.getConfig()
	const USER_BOUND_STRING = "${YOUR_USER_BOUND_VALUE}"
	// TODO encrypt mattermostUserID?
	connectURL := strings.Replace(
		conf.JiraAuthAppURL, USER_BOUND_STRING, mattermostUserID, 1)
	return connectURL, nil, nil
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
