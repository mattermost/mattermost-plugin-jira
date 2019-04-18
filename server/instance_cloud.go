// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	ajwt "github.com/rbriski/atlassian-jwt"
	"golang.org/x/oauth2"
	oauth2_jira "golang.org/x/oauth2/jira"
)

type jiraCloudInstance struct {
	*JIRAInstance

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"none"`
	oauth2Config                oauth2.Config `json:"none"`
}

var _ Instance = (*jiraCloudInstance)(nil)

type AtlassianSecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	PublicKey      string `json:"publicKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
	OAuthClientId  string `json:"oauthClientId"`
}

func NewJIRACloudInstance(p *Plugin, key, rawASC string, asc *AtlassianSecurityContext) Instance {
	return &jiraCloudInstance{
		JIRAInstance:                NewJIRAInstance(p, JIRATypeCloud, key),
		RawAtlassianSecurityContext: rawASC,
		AtlassianSecurityContext:    asc,
	}
}

func (jci jiraCloudInstance) InitWithPlugin(p *Plugin) Instance {
	return NewJIRACloudInstance(p, jci.JIRAInstance.Key, jci.RawAtlassianSecurityContext, jci.AtlassianSecurityContext)
}

func (jci jiraCloudInstance) GetUserConnectURL(p *Plugin, mattermostUserId string) (string, error) {
	token, err := p.NewEncodedAuthToken(mattermostUserId)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Add(argMMToken, token)
	return fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/mattermost-plugin/user-config?%v",
		jci.GetURL(), jci.GetURL(), v.Encode()), nil
}

func (jci jiraCloudInstance) GetURL() string {
	return jci.AtlassianSecurityContext.BaseURL
}

func (jci jiraCloudInstance) GetJIRAClient(jiraUser JIRAUser) (*jira.Client, error) {
	client, _, err := jci.getJIRAClientForUser(jiraUser)
	if err == nil {
		return client, nil
	}

	client, err = jci.getJIRAClientForServer()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Creates a client for acting on behalf of a user
func (jci jiraCloudInstance) getJIRAClientForUser(jiraUser JIRAUser) (*jira.Client, *http.Client, error) {
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

func (jci jiraCloudInstance) parseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error) {
	r.ParseForm()
	tokenString := r.Form.Get("jwt")
	if tokenString == "" {
		return nil, "", errors.New("jwt not found in the request")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// HMAC secret is a []byte
		return []byte(jci.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil {
		return nil, "", err
	}

	return token, tokenString, nil
}
