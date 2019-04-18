// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

type jiraServerInstance struct {
	*JIRAInstance

	JIRAServerURL string

	oauth1Config *oauth1.Config `json:"none"`
}

var _ Instance = (*jiraServerInstance)(nil)

func NewJIRAServerInstance(p *Plugin, jiraURL string) Instance {
	return &jiraServerInstance{
		JIRAInstance:  NewJIRAInstance(p, JIRATypeServer, jiraURL),
		JIRAServerURL: jiraURL,
	}
}

func (jsi jiraServerInstance) InitWithPlugin(p *Plugin) Instance {
	return NewJIRAServerInstance(p, jsi.JIRAServerURL)
}

func (jis jiraServerInstance) GetURL() string {
	return jis.JIRAServerURL
}

func (jis jiraServerInstance) GetUserConnectURL(p *Plugin, mattermostUserId string) (string, error) {
	oauth1Config, err := jis.GetOAuth1Config()
	if err != nil {
		return "", err
	}

	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", err
	}

	err = p.StoreOAuth1RequestToken(token, secret)
	if err != nil {
		return "", err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", err
	}

	return authURL.String(), nil
}

func (jis jiraServerInstance) GetJIRAClient(jiraUser JIRAUser) (*jira.Client, error) {
	if jiraUser.Oauth1AccessToken == "" || jiraUser.Oauth1AccessSecret == "" {
		return nil, errors.New("No access token, please use /jira connect")
	}

	oauth1Config, err := jis.GetOAuth1Config()
	if err != nil {
		return nil, errors.WithMessage(err, "could not get oauth1 config")
	}

	token := oauth1.NewToken(jiraUser.Oauth1AccessToken, jiraUser.Oauth1AccessSecret)
	httpClient := oauth1Config.Client(oauth1.NoContext, token)
	jiraClient, err := jira.NewClient(httpClient, jis.GetURL())
	if err != nil {
		return nil, errors.WithMessage(err, "could not get jira client")
	}

	return jiraClient, nil
}

func (jis jiraServerInstance) ParseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error) {
	return nil, "", fmt.Errorf("NOT IMPLEMENTED: ParseHTTPRequestJWT")
}

func (jis jiraServerInstance) getOAuth1Config() *oauth1.Config {
	jis.lock.RLock()
	defer jis.lock.RUnlock()

	return jis.oauth1Config
}

func (jis *jiraServerInstance) GetOAuth1Config() (*oauth1.Config, error) {
	oauth1Config := jis.getOAuth1Config()
	if oauth1Config != nil {
		return oauth1Config, nil
	}

	rsaKey, err := jis.EnsureRSAKey()
	if err != nil {
		return nil, err
	}

	jis.lock.Lock()
	defer jis.lock.Unlock()
	jis.oauth1Config = &oauth1.Config{
		// TODO make these configurable
		ConsumerKey:    "ConsumerKey",
		ConsumerSecret: "dontcare",
		CallbackURL:    fmt.Sprintf("%v/oauth1/complete", jis.GetPluginURL()),
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: jis.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    jis.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  jis.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
	}

	return jis.oauth1Config, nil
}
