package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type cloudOAuthInstance struct {
	*InstanceCommon

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	JiraResourceID string
}

type JiraAccessibleResources []struct {
	Id string
}

var _ Instance = (*cloudOAuthInstance)(nil)

const JIRA_SCOPES = "read:me,read:jira-work,write:jira-work"

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
	oauth2Conf := ci.GetOAuthConfig()

	ctx := context.Background()
	tokenSource := oauth2Conf.TokenSource(ctx, connection.OAuth2Token)
	client := oauth2.NewClient(ctx, tokenSource)

	// TODO Get resource ID if not in the KV Store?
	jiraID, err := ci.getJiraCloudResourceID(*client)
	ci.JiraResourceID = jiraID
	if err != nil {
		return nil, nil, err
	}

	jiraClient, err := jira.NewClient(client, ci.GetURL())
	return jiraClient, client, err
}

func (ci *cloudOAuthInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Jira Cloud Mattermost Key": ci.MattermostKey,
	}
}

func (ci *cloudOAuthInstance) GetUserConnectURL(mattermostUserID string) (string, *http.Cookie, error) {
	oauthConf := ci.GetOAuthConfig()
	url := oauthConf.AuthCodeURL(
		ci.generateRandomState(),
		oauth2.SetAuthURLParam("audience", "api.atlassian.com"),
		oauth2.SetAuthURLParam("state", mattermostUserID),
		oauth2.SetAuthURLParam("response_type", "code"),
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	return url, nil, nil
}

func (ci *cloudOAuthInstance) generateRandomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (ci *cloudOAuthInstance) GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		Scopes:       strings.Split(JIRA_SCOPES, ","),
		RedirectURL:  fmt.Sprintf("%s/oauth/connect", ci.Plugin.GetPluginURL()),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.com/authorize",
			TokenURL: "https://auth.atlassian.com/oauth/token",
		},
	}
}

func (ci *cloudOAuthInstance) GetURL() string {
	return "https://api.atlassian.com/ex/jira/" + ci.JiraResourceID
}

func (ci *cloudOAuthInstance) GetManageAppsURL() string {
	// TODO
	return fmt.Sprintf("%s/plugins/servlet/applinks/listApplicationLinks", ci.GetURL())
}

func (ci *cloudOAuthInstance) GetManageWebhooksURL() string {
	// TODO
	return fmt.Sprintf("%s/plugins/servlet/webhooks", ci.GetURL())
}

func (ci *cloudOAuthInstance) GetMattermostKey() string {
	return ci.MattermostKey
}

func (ci *cloudOAuthInstance) getJiraCloudResourceID(client http.Client) (string, error) {
	request, err := http.NewRequest(
		"GET",
		"https://api.atlassian.com/oauth/token/accessible-resources",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed getting request")
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed getting accessible resources: %s", err.Error())
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed read accesible resources response: %s", err.Error())
	}

	var resources JiraAccessibleResources
	err = json.Unmarshal(contents, &resources)

	if err != nil {
		return "", fmt.Errorf("failed marshall: %s", err.Error())
	}

	// We return the first resource ID only
	if len(resources) < 1 {
		return "", errors.New("No resources available for this Jira Cloud Account")
	}

	return resources[0].Id, nil
}
