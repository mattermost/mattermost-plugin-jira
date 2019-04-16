// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	jwt "github.com/dgrijalva/jwt-go"
)

type jiraServerInstance struct {
	jiraInstance
	lock *sync.RWMutex

	JIRAServerURL string

	oauth1Config *oauth1.Config `json:"none"`
}

var _ JIRAInstance = (*jiraServerInstance)(nil)

func NewJIRAServerInstance(p *Plugin, jiraURL string) JIRAInstance {
	return &jiraServerInstance{
		jiraInstance: jiraInstance{
			Plugin: p,
			Type:   JIRAServerType,
			Key:    jiraURL,
		},
		lock:          &sync.RWMutex{},
		JIRAServerURL: jiraURL,
	}
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

func (jis jiraServerInstance) GetJIRAClientForUser(info JIRAUserInfo) (*jira.Client, *http.Client, error) {
	return nil, nil, fmt.Errorf("NOT IMPLEMENTED: GetJIRAClientForUser")
}

func (jis jiraServerInstance) GetJIRAClientForServer() (*jira.Client, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED: GetJIRAClientForServer")
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
