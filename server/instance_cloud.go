// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	ajwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"
)

type jiraCloudInstance struct {
	*JIRAInstance

	// Initially a new instance is created with an expiration time. The
	// admin is expected to upload it to the Jira instance, and we will
	// then receive a /installed callback that initializes the instance
	// and makes it permanent. No subsequent /installed will be accepted
	// for the instance.
	Installed bool

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"none"`
}

var _ Instance = (*jiraCloudInstance)(nil)

type AtlassianSecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
	OAuthClientId  string `json:"oauthClientId"`
}

func NewJIRACloudInstance(key string, installed bool, rawASC string,
	asc *AtlassianSecurityContext) *jiraCloudInstance {

	return &jiraCloudInstance{
		JIRAInstance:                NewJIRAInstance(JIRATypeCloud, key),
		Installed:                   installed,
		RawAtlassianSecurityContext: rawASC,
		AtlassianSecurityContext:    asc,
	}
}

func (jci jiraCloudInstance) GetMattermostKey() string {
	return jci.AtlassianSecurityContext.Key
}

func (jci jiraCloudInstance) GetDisplayDetails() map[string]string {
	if !jci.Installed {
		return map[string]string{
			"Setup": "In progress",
		}
	}

	return map[string]string{
		"Key":            jci.AtlassianSecurityContext.Key,
		"ClientKey":      jci.AtlassianSecurityContext.ClientKey,
		"ServerVersion":  jci.AtlassianSecurityContext.ServerVersion,
		"PluginsVersion": jci.AtlassianSecurityContext.PluginsVersion,
	}
}

func (jci jiraCloudInstance) GetUserConnectURL(a *Action) (string, error) {
	secret := make([]byte, 256)
	_, err := rand.Read(secret)
	if err != nil {
		return "", err
	}
	secretKey := fmt.Sprintf("%x", sha256.Sum256(secret))
	secretValue := "true"
	err = a.Plugin.StoreOneTimeSecret(secretKey, secretValue)
	if err != nil {
		return "", err
	}

	token, err := NewEncodedAuthToken(a.Plugin, a.MattermostUserId, secretKey)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Add(argMMToken, token)
	return fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/%s/%s?%v",
		jci.GetURL(), jci.GetURL(), jci.AtlassianSecurityContext.Key, userRedirectPageKey, v.Encode()), nil
}

func (jci jiraCloudInstance) GetURL() string {
	return jci.AtlassianSecurityContext.BaseURL
}

func (jci jiraCloudInstance) GetJIRAClient(a *Action, jiraUser *JIRAUser) (*jira.Client, error) {
	if jiraUser == nil {
		jiraUser = a.JiraUser
	}
	client, _, err := jci.getJIRAClientForUser(a, jiraUser)
	if err == nil {
		return client, nil
	}

	//TODO decide if we ever need this as the default client
	// client, err = jci.getJIRAClientForServer()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get Jira client for user "+a.JiraUser.Name)
	}

	return client, nil
}

// Creates a client for acting on behalf of a user
func (jci jiraCloudInstance) getJIRAClientForUser(a *Action, jiraUser *JIRAUser) (*jira.Client, *http.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: jci.GetURL(),
		Subject: jiraUser.Name,
	}

	oauth2Conf.Config.ClientID = jci.AtlassianSecurityContext.OAuthClientId
	oauth2Conf.Config.ClientSecret = jci.AtlassianSecurityContext.SharedSecret
	oauth2Conf.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	oauth2Conf.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := oauth2Conf.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, oauth2Conf.BaseURL)
	return jiraClient, httpClient, err
}

// Creates a "bot" client with a JWT
func (jci jiraCloudInstance) getJIRAClientForServer() (*jira.Client, error) {
	jwtConf := &ajwt.Config{
		Key:          jci.AtlassianSecurityContext.Key,
		ClientKey:    jci.AtlassianSecurityContext.ClientKey,
		SharedSecret: jci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      jci.AtlassianSecurityContext.BaseURL,
	}

	return jira.NewClient(jwtConf.Client(), jwtConf.BaseURL)
}
