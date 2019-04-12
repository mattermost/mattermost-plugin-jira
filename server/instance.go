// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	ajwt "github.com/rbriski/atlassian-jwt"
	"golang.org/x/oauth2"
	oauth2_jira "golang.org/x/oauth2/jira"
)

const (
	JIRACloudType  = "cloud"
	JIRAServerType = "server"
)

const prefixForInstance = true

type JIRAInstance struct {
	Key string

	// One of JIRAxxxType
	Type string

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"none"`
	oauth2Config                oauth2.Config `json:"none"`

	// For server instances
	JIRAServerURL string
	oauth1Config  oauth1.Config `json:"none"`
}

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
	ji := JIRAInstance{
		Type:                        JIRACloudType,
		Key:                         key,
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
	ji.oauth2Config = oauth2.Config{
		// ClientID:     "LimAAPOhX7ncIN7cPB77tZ1Gwz0r2WmL",
		// ClientSecret: "01_Y6g1JRmLnSGcaRU19LzhfnsXHAGwtuQTacQscxR3eCy7tzhLYYbuQHXiVIJq_",
		// Scopes:       []string{"read:jira-work", "read:jira-user", "write:jira-work"},
		// Endpoint: oauth2.Endpoint{
		// 	AuthURL:  "https://auth.atlassian.com/authorize",
		// 	TokenURL: "https://auth.atlassian.com/oauth/token",
		// },
		// RedirectURL: fmt.Sprintf("%v/plugins/%v/oauth/complete", p.externalURL(), manifest.Id),
	}

	return ji
}

func NewJIRAServerInstance(jiraURL, mmURL string, rsaKey *rsa.PrivateKey) JIRAInstance {
	ji := JIRAInstance{
		Type:          JIRAServerType,
		Key:           jiraURL,
		JIRAServerURL: jiraURL,
		oauth1Config: oauth1.Config{
			ConsumerKey:    "mm-token",
			ConsumerSecret: "dont_care",
			CallbackURL:    fmt.Sprintf("%v/plugins/%v/oauth1/complete", mmURL, manifest.Id),
			Endpoint: oauth1.Endpoint{
				RequestTokenURL: jiraURL + "/plugins/servlet/oauth/request-token",
				AuthorizeURL:    jiraURL + "/plugins/servlet/oauth/authorize",
				AccessTokenURL:  jiraURL + "/plugins/servlet/oauth/access-token",
			},
			Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
		},
	}

	return ji
}

func (ji JIRAInstance) isEmpty() bool {
	return len(ji.Key) == 0
}

func (ji JIRAInstance) URL() string {
	switch ji.Type {
	case JIRACloudType:
		return ji.AtlassianSecurityContext.BaseURL
	case JIRAServerType:
		return ji.JIRAServerURL
	}
	return ""
}

func (ji JIRAInstance) GetJIRAClientForUser(info JIRAUserInfo) (*jira.Client, *http.Client, error) {
	switch ji.Type {
	case JIRACloudType:
		return ji.getJIRACloudClientForUser(info.Name)
		// case JIRAServerType:
		// 	return ji.getJIRAServerClientForUser(info.AccountId)
	}
	return nil, nil, fmt.Errorf("Invalid instance type: %s", ji.Type)
}

// Creates a client for acting on behalf of a user
func (ji JIRAInstance) getJIRACloudClientForUser(jiraUser string) (*jira.Client, *http.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: ji.URL(),
		Subject: jiraUser,
	}

	oauth2Conf.Config.ClientID = ji.AtlassianSecurityContext.OAuthClientId
	oauth2Conf.Config.ClientSecret = ji.AtlassianSecurityContext.SharedSecret
	oauth2Conf.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	oauth2Conf.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := oauth2Conf.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, oauth2Conf.BaseURL)
	return jiraClient, httpClient, err
}

// Creates a "bot" client with a JWT
func (ji JIRAInstance) getJIRACloudClientForServer() (*jira.Client, error) {
	jwtConf := &ajwt.Config{
		Key:          ji.AtlassianSecurityContext.Key,
		ClientKey:    ji.AtlassianSecurityContext.ClientKey,
		SharedSecret: ji.AtlassianSecurityContext.SharedSecret,
		BaseURL:      ji.AtlassianSecurityContext.BaseURL,
	}

	return jira.NewClient(jwtConf.Client(), jwtConf.BaseURL)
}

func (ji JIRAInstance) parseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error) {
	if ji.Type != JIRACloudType {
		return nil, "", errors.New("not supported for " + ji.Type)
	}

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
		return []byte(ji.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil {
		return nil, "", err
	}

	return token, tokenString, nil
}
