// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"

	jira "github.com/andygrunwald/go-jira"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	ajwt "github.com/rbriski/atlassian-jwt"
	"golang.org/x/oauth2"
	oauth2_jira "golang.org/x/oauth2/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type cloudInstance struct {
	*InstanceCommon

	// Initially a new instance is created with an expiration time. The
	// admin is expected to upload it to the Jira instance, and we will
	// then receive a /installed callback that initializes the instance
	// and makes it permanent. No subsequent /installed will be accepted
	// for the instance.
	Installed bool

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"-"`
}

var _ Instance = (*cloudInstance)(nil)

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

func newCloudInstance(p *Plugin, key types.ID, installed bool, rawASC string, asc *AtlassianSecurityContext) *cloudInstance {
	return &cloudInstance{
		InstanceCommon:              newInstanceCommon(p, CloudInstanceType, key),
		Installed:                   installed,
		RawAtlassianSecurityContext: rawASC,
		AtlassianSecurityContext:    asc,
	}
}

func (ci *cloudInstance) GetMattermostKey() string {
	return ci.AtlassianSecurityContext.Key
}

func (ci *cloudInstance) GetDisplayDetails() map[string]string {
	if !ci.Installed {
		return map[string]string{
			"Setup": "In progress",
		}
	}

	return map[string]string{
		"Atlassian Connect Key":        ci.AtlassianSecurityContext.Key,
		"Atlassian Connect Client Key": ci.AtlassianSecurityContext.ClientKey,
		"Jira Cloud Version":           ci.AtlassianSecurityContext.ServerVersion,
		"Jira Cloud Plugins Version":   ci.AtlassianSecurityContext.PluginsVersion,
	}
}

func (ci *cloudInstance) GetUserConnectURL(mattermostUserId string) (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	secret := fmt.Sprintf("%x", randomBytes)
	err = ci.Plugin.otsStore.StoreOneTimeSecret(mattermostUserId, secret)
	if err != nil {
		return "", err
	}

	token, err := ci.Plugin.NewEncodedAuthToken(mattermostUserId, secret)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Add(argMMToken, token)
	connectURL := fmt.Sprintf("%s/login?dest-url=%s/plugins/servlet/ac/%s/%s?%v",
		ci.GetURL(), ci.GetURL(), ci.AtlassianSecurityContext.Key, userRedirectPageKey, v.Encode())
	return connectURL, nil
}

func (ci *cloudInstance) GetURL() string {
	return ci.AtlassianSecurityContext.BaseURL
}

func (ci *cloudInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/plugins/servlet/upm", ci.GetURL())
}

func (ci *cloudInstance) GetManageWebhooksURL() string {
	return fmt.Sprintf("%s/plugins/servlet/webhooks", ci.GetURL())
}

func (ci *cloudInstance) GetClient(connection *Connection) (Client, error) {
	client, _, err := ci.getClientForConnection(connection)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get Jira client for user "+connection.DisplayName)
	}
	return newCloudClient(client), nil
}

// Creates a client for acting on behalf of a user
func (ci *cloudInstance) getClientForConnection(connection *Connection) (*jira.Client, *http.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: ci.GetURL(),
		Subject: connection.AccountID,
		Config: oauth2.Config{
			ClientID:     ci.AtlassianSecurityContext.OAuthClientId,
			ClientSecret: ci.AtlassianSecurityContext.SharedSecret,
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
	httpClient = expvar.WrapHTTPClient(httpClient,
		conf.stats, endpointNameFromRequest)

	jiraClient, err := jira.NewClient(httpClient, oauth2Conf.BaseURL)
	return jiraClient, httpClient, err
}

// Creates a "bot" client with a JWT
func (ci *cloudInstance) getClientForBot() (*jira.Client, error) {
	conf := ci.getConfig()
	jwtConf := &ajwt.Config{
		Key:          ci.AtlassianSecurityContext.Key,
		ClientKey:    ci.AtlassianSecurityContext.ClientKey,
		SharedSecret: ci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      ci.AtlassianSecurityContext.BaseURL,
	}

	httpClient := jwtConf.Client()
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))
	httpClient = expvar.WrapHTTPClient(httpClient,
		conf.stats, endpointNameFromRequest)

	return jira.NewClient(httpClient, jwtConf.BaseURL)
}

func (ci *cloudInstance) parseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, "", errors.WithMessage(err, "failed to parse request")
	}
	tokenString := r.FormValue("jwt")
	if tokenString == "" {
		return nil, "", errors.New("no jwt in the request")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(
				fmt.Sprintf("unsupported signing method: %v", token.Header["alg"]))
		}
		// HMAC secret is a []byte
		return []byte(ci.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, "", errors.WithMessage(err, "failed to validate JWT")
	}

	return token, tokenString, nil
}
