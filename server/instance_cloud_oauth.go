package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type cloudOAuthInstance struct {
	*InstanceCommon

	// The SiteURL may change as we go, so we store the PluginKey when it was installed
	MattermostKey string

	JiraResourceID   string
	JiraClientID     string
	JiraClientSecret string
	JiraBaseURL      string
	CodeVerifier     string
	CodeChallenge    string
	JWTInstance      *cloudInstance
}

type CloudOAuthConfigure struct {
	InstanceURL  string `json:"instance_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type JiraAccessibleResources []struct {
	ID string
}

type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
}

var _ Instance = (*cloudOAuthInstance)(nil)
var jiraOAuthAccessibleResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"

const (
	JiraScopes          = "read:jira-user,read:jira-work,write:jira-work"
	JiraScopesOffline   = JiraScopes + ",offline_access"
	JiraResponseType    = "code"
	JiraConsent         = "consent"
	PKCEByteArrayLength = 32
)

func (p *Plugin) installCloudOAuthInstance(rawURL string) (string, *cloudOAuthInstance, error) {
	jiraURL, err := utils.CheckJiraURL(p.GetSiteURL(), rawURL, false)
	if err != nil {
		return "", nil, err
	}
	if !utils.IsJiraCloudURL(jiraURL) {
		return "", nil, errors.Errorf("`%s` is a Jira server URL instead of a Jira Cloud URL", jiraURL)
	}

	params, err := getS256PKCEParams()
	if err != nil {
		return "", nil, err
	}

	newInstance := &cloudOAuthInstance{
		InstanceCommon: newInstanceCommon(p, CloudOAuthInstanceType, types.ID(jiraURL)),
		MattermostKey:  p.GetPluginKey(),
		JiraBaseURL:    rawURL,
		CodeVerifier:   params.CodeVerifier,
		CodeChallenge:  params.CodeChallenge,
	}

	existingInstance, err := p.instanceStore.LoadInstance(types.ID(jiraURL))
	if err != nil && !errors.Is(err, kvstore.ErrNotFound) {
		return "", nil, errors.Wrapf(err, "failed to load existing jira instance. ID: %s", jiraURL)
	}

	// Handle backwards compatibility with existing JWT instance
	if existingInstance != nil {
		if existingInstance.Common().Type == CloudOAuthInstanceType {
			oauthInstance, ok := existingInstance.(*cloudOAuthInstance)
			if !ok {
				return "", nil, errors.Wrapf(err, "failed to assert existing cloud-oauth instance as cloudOAuthInstance. ID: %s", jiraURL)
			}

			newInstance.JWTInstance = oauthInstance.JWTInstance
			if newInstance.JWTInstance != nil {
				p.API.LogDebug("Installing cloud-oauth over existing cloud-oauth instance. Carrying over existing saved JWT instance.")
			} else {
				p.API.LogDebug("Installing cloud-oauth over existing cloud-oauth instance. There exists no previous JWT instance to carry over.")
			}
		} else if existingInstance.Common().Type == CloudInstanceType {
			jwtInstance, ok := existingInstance.(*cloudInstance)
			if !ok {
				return "", nil, errors.Wrapf(err, "failed to assert existing cloud instance as cloudInstance. ID: %s", jiraURL)
			}

			newInstance.JWTInstance = jwtInstance
			p.API.LogDebug("Installing cloud-oauth over existing cloud JWT instance. Carrying over existing saved JWT instance.")
		}
	} else {
		p.API.LogDebug("Installing new cloud-oauth instance. There exists no previous JWT instance to carry over.")
	}

	if err = p.InstallInstance(newInstance); err != nil {
		return "", nil, errors.Wrapf(err, "failed to install cloud-oauth instance. ID: %s", jiraURL)
	}

	return jiraURL, newInstance, nil
}

func (ci *cloudOAuthInstance) GetClient(connection *Connection) (Client, error) {
	client, _, err := ci.getClientForConnection(connection)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to get Jira client for the user %s", connection.DisplayName))
	}
	return newCloudClient(client), nil
}

func (ci *cloudOAuthInstance) getClientForConnection(connection *Connection) (*jira.Client, *http.Client, error) {
	oauth2Conf := ci.GetOAuthConfig()
	ctx := context.Background()

	// Checking if this user's connection is for a JWT instance
	if connection.OAuth2Token == nil {
		if ci.JWTInstance != nil {
			ci.Plugin.API.LogDebug("Returning a JWT token client since the stored JWT instance is not nil and the user's oauth token is nil")
			return ci.JWTInstance.getClientForConnection(connection)
		}

		return nil, nil, errors.New("failed to create client for OAuth instance: no JWT instance found, and connection's OAuth token is missing")
	}

	tokenSource := oauth2Conf.TokenSource(ctx, connection.OAuth2Token)
	client := oauth2.NewClient(ctx, tokenSource)

	// Get a new token, if Access Token has expired
	currentToken := connection.OAuth2Token
	updatedToken, err := tokenSource.Token()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error in getting token from token source")
	}

	if updatedToken.RefreshToken != currentToken.RefreshToken {
		connection.OAuth2Token = updatedToken

		// Store this new access token & refresh token to get a new access token in future when it has expired
		if err = ci.Plugin.userStore.StoreConnection(ci.Common().InstanceID, connection.MattermostUserID, connection); err != nil {
			return nil, nil, err
		}
	}

	// TODO: Get resource ID if not in the KV Store?
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
	state := fmt.Sprintf("%s_%s", model.NewId()[0:15], mattermostUserID)
	url := oauthConf.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("audience", "api.atlassian.com"),
		oauth2.SetAuthURLParam("state", state),
		oauth2.SetAuthURLParam("response_type", "code"),
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", ci.CodeChallenge),
	)
	if err := ci.Plugin.otsStore.StoreOneTimeSecret(mattermostUserID, state); err != nil {
		return "", nil, err
	}
	return url, nil, nil
}

func (ci *cloudOAuthInstance) GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ci.JiraClientID,
		ClientSecret: ci.JiraClientSecret,
		Scopes:       strings.Split(JiraScopesOffline, ","),
		RedirectURL:  fmt.Sprintf("%s%s", ci.Plugin.GetPluginURL(), instancePath(routeOAuth2Complete, ci.InstanceID)),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.com/authorize",
			TokenURL: "https://auth.atlassian.com/oauth/token",
		},
	}
}

func (ci *cloudOAuthInstance) GetURL() string {
	return "https://api.atlassian.com/ex/jira/" + ci.JiraResourceID
}

func (ci *cloudOAuthInstance) GetJiraBaseURL() string {
	return ci.JiraBaseURL
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

func (ci *cloudOAuthInstance) getJiraCloudResourceID(client http.Client) (string, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		jiraOAuthAccessibleResourcesURL,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get the request")
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to get the accessible resources: %s", err.Error())
	}

	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read accessible resources response: %s", err.Error())
	}

	var resources JiraAccessibleResources
	if err = json.Unmarshal(contents, &resources); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal JiraAccessibleResources")
	}

	// We return the first resource ID only
	if len(resources) < 1 {
		return "", errors.New("No resources are available for this Jira Cloud Account.")
	}

	return resources[0].ID, nil
}
