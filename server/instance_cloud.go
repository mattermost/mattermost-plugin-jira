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
	jiraInstance

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"none"`
	oauth2Config                oauth2.Config `json:"none"`
}

var _ JIRAInstance = (*jiraCloudInstance)(nil)

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

func NewJIRACloudInstance(key, rawASC string, asc *AtlassianSecurityContext) JIRAInstance {
	jci := &jiraCloudInstance{
		jiraInstance: jiraInstance{
			Type: JIRACloudType,
			Key:  key,
		},
		RawAtlassianSecurityContext: rawASC,
		AtlassianSecurityContext:    asc,
	}

	// This was experimental. There is a way to define an application on
	// dev.atlassian.com, and have valid OAuth2 credentials issued, that could then
	// be cut & pasted into config values and used. However, the application would
	// then have a singular OAuth2 callback URL configured in the dev.atlassian.com
	// web site, essenttially requiring thatt a separate application is defined for
	// each relevant Mattermost instance. I was not able to make OAuth2 work with
	// credentials obtained from atlassian-connect.json /installed callback.
	//
	// Keeping the code for now, in case something changes.
	jci.oauth2Config = oauth2.Config{
		// ClientID:     "LimAAPOhX7ncIN7cPB77tZ1Gwz0r2WmL",
		// ClientSecret: "01_Y6g1JRmLnSGcaRU19LzhfnsXHAGwtuQTacQscxR3eCy7tzhLYYbuQHXiVIJq_",
		// Scopes:       []string{"read:jira-work", "read:jira-user", "write:jira-work"},
		// Endpoint: oauth2.Endpoint{
		// 	AuthURL:  "https://auth.atlassian.com/authorize",
		// 	TokenURL: "https://auth.atlassian.com/oauth/token",
		// },
		// RedirectURL: fmt.Sprintf("%v/plugins/%v/oauth/complete", p.externalURL(), manifest.Id),
	}

	return jci
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

// Creates a client for acting on behalf of a user
func (jci jiraCloudInstance) GetJIRAClientForUser(info JIRAUserInfo) (*jira.Client, *http.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: jci.GetURL(),
		Subject: info.Name,
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
func (jci jiraCloudInstance) GetJIRAClientForServer() (*jira.Client, error) {
	jwtConf := &ajwt.Config{
		Key:          jci.AtlassianSecurityContext.Key,
		ClientKey:    jci.AtlassianSecurityContext.ClientKey,
		SharedSecret: jci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      jci.AtlassianSecurityContext.BaseURL,
	}

	return jira.NewClient(jwtConf.Client(), jwtConf.BaseURL)
}

func (jci jiraCloudInstance) ParseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error) {
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
