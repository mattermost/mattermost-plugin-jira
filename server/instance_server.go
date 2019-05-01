// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"
)

type jiraServerInstance struct {
	*JIRAInstance

	JIRAServerURL string

	oauth1Config *oauth1.Config
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

func (jsi jiraServerInstance) GetURL() string {
	return jsi.JIRAServerURL
}

func (jsi jiraServerInstance) GetUserConnectURL(p *Plugin, mattermostUserId string) (returnURL string, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
	}()

	oauth1Config, err := jsi.GetOAuth1Config()
	if err != nil {
		return "", err
	}

	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", err
	}

	err = p.StoreOneTimeSecret(token, secret)
	if err != nil {
		return "", err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", err
	}

	return authURL.String(), nil
}

func (jsi jiraServerInstance) GetJIRAClient(jiraUser JIRAUser) (returnClient *jira.Client, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a Jira client for "+jiraUser.Name)
	}()

	if jiraUser.Oauth1AccessToken == "" || jiraUser.Oauth1AccessSecret == "" {
		return nil, errors.New("No access token, please use /jira connect")
	}

	oauth1Config, err := jsi.GetOAuth1Config()
	if err != nil {
		return nil, err
	}

	token := oauth1.NewToken(jiraUser.Oauth1AccessToken, jiraUser.Oauth1AccessSecret)
	httpClient := oauth1Config.Client(oauth1.NoContext, token)
	jiraClient, err := jira.NewClient(httpClient, jsi.GetURL())
	if err != nil {
		return nil, err
	}

	return jiraClient, nil
}

func (jsi jiraServerInstance) getOAuth1Config() *oauth1.Config {
	jsi.lock.RLock()
	defer jsi.lock.RUnlock()

	return jsi.oauth1Config
}

func (jsi *jiraServerInstance) GetOAuth1Config() (returnConfig *oauth1.Config, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to create an OAuth1 config")
	}()

	oauth1Config := jsi.getOAuth1Config()
	if oauth1Config != nil {
		return oauth1Config, nil
	}

	rsaKey, err := jsi.EnsureRSAKey()
	if err != nil {
		return nil, err
	}

	jsi.lock.Lock()
	defer jsi.lock.Unlock()
	jsi.oauth1Config = &oauth1.Config{
		// TODO make these configurable
		ConsumerKey:    "ConsumerKey",
		ConsumerSecret: "dontcare",
		CallbackURL:    fmt.Sprintf("%v/oauth1/complete", jsi.GetPluginURL()),
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: jsi.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    jsi.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  jsi.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
	}

	return jsi.oauth1Config, nil
}
