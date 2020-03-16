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
	*AtlassianSecurityContext   `json:"-"`
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

func NewJIRACloudInstance(p *Plugin, key string, installed bool, rawASC string, asc *AtlassianSecurityContext) Instance {
	return &jiraCloudInstance{
		JIRAInstance:                NewJIRAInstance(p, JIRATypeCloud, key),
		Installed:                   installed,
		RawAtlassianSecurityContext: rawASC,
		AtlassianSecurityContext:    asc,
	}
}

type withCloudInstanceFunc func(jci *jiraCloudInstance, w http.ResponseWriter, r *http.Request) (int, error)

func withCloudInstance(p *Plugin, w http.ResponseWriter, r *http.Request, f withCloudInstanceFunc) (int, error) {
	return withInstance(p.currentInstanceStore, w, r, func(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
		jci, ok := ji.(*jiraCloudInstance)
		if !ok {
			return respondErr(w, http.StatusBadRequest, errors.New("Must be a JIRA Cloud instance, is "+ji.GetType()))
		}
		return f(jci, w, r)
	})
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
		"Atlassian Connect Key":        jci.AtlassianSecurityContext.Key,
		"Atlassian Connect Client Key": jci.AtlassianSecurityContext.ClientKey,
		"Jira Cloud Version":           jci.AtlassianSecurityContext.ServerVersion,
		"Jira Cloud Plugins Version":   jci.AtlassianSecurityContext.PluginsVersion,
	}
}

func (jci jiraCloudInstance) GetUserConnectURL(mattermostUserId string) (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	secret := fmt.Sprintf("%x", randomBytes)
	err = jci.Plugin.otsStore.StoreOneTimeSecret(mattermostUserId, secret)
	if err != nil {
		return "", err
	}

	token, err := jci.Plugin.NewEncodedAuthToken(mattermostUserId, secret)
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

func (jci jiraCloudInstance) GetClient(jiraUser JIRAUser) (Client, error) {
	client, _, err := jci.getJIRAClientForUser(jiraUser)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get Jira client for user "+jiraUser.DisplayName)
	}
	return newCloudClient(client), nil
}

// Creates a client for acting on behalf of a user
func (jci jiraCloudInstance) getJIRAClientForUser(jiraUser JIRAUser) (*jira.Client, *http.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: jci.GetURL(),
		Subject: jiraUser.AccountID,
		Config: oauth2.Config{
			ClientID:     jci.AtlassianSecurityContext.OAuthClientId,
			ClientSecret: jci.AtlassianSecurityContext.SharedSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://auth.atlassian.io",
				TokenURL: "https://auth.atlassian.io/oauth2/token",
			},
		},
	}

	conf := jci.GetPlugin().getConfig()
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
func (jci jiraCloudInstance) getJIRAClientForBot() (*jira.Client, error) {
	conf := jci.GetPlugin().getConfig()
	jwtConf := &ajwt.Config{
		Key:          jci.AtlassianSecurityContext.Key,
		ClientKey:    jci.AtlassianSecurityContext.ClientKey,
		SharedSecret: jci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      jci.AtlassianSecurityContext.BaseURL,
	}

	httpClient := jwtConf.Client()
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))
	httpClient = expvar.WrapHTTPClient(httpClient,
		conf.stats, endpointNameFromRequest)

	return jira.NewClient(httpClient, jwtConf.BaseURL)
}

func (jci jiraCloudInstance) parseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error) {
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
		return []byte(jci.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, "", errors.WithMessage(err, "failed to validate JWT")
	}

	return token, tokenString, nil
}
