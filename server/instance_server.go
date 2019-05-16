// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"
)

type jiraServerInstance struct {
	*JIRAInstance

	JIRAServerURL string

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	oauth1Config *oauth1.Config
}

var _ Instance = (*jiraServerInstance)(nil)

func NewJIRAServerInstance(jiraURL, mattermostKey string) Instance {
	return &jiraServerInstance{
		JIRAInstance:  NewJIRAInstance(JIRATypeServer, jiraURL),
		MattermostKey: mattermostKey,
		JIRAServerURL: jiraURL,
	}
}

func (jsi jiraServerInstance) GetURL() string {
	return jsi.JIRAServerURL
}

type withServerInstanceFunc func(jsi *jiraServerInstance, w http.ResponseWriter, r *http.Request) (int, error)

func (jsi jiraServerInstance) GetMattermostKey() string {
	return jsi.MattermostKey
}

func (jsi jiraServerInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": jsi.MattermostKey,
	}
}

func (jsi jiraServerInstance) GetUserConnectURL(a *Action) (returnURL string, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
		}
	}()

	oauth1Config, err := jsi.GetOAuth1Config(a)
	if err != nil {
		return "", err
	}

	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", err
	}

	err = a.Plugin.StoreOneTimeSecret(token, secret)
	if err != nil {
		return "", err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", err
	}

	return authURL.String(), nil
}

func (jsi jiraServerInstance) GetJIRAClient(a *Action, jiraUser *JIRAUser) (
	returnClient *jira.Client, returnErr error) {

	if jiraUser == nil {
		jiraUser = a.JiraUser
	}

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessagef(returnErr,
				"failed to get a Jira client for %q", jiraUser.Name)
		}
	}()

	if jiraUser.Oauth1AccessToken == "" || jiraUser.Oauth1AccessSecret == "" {
		return nil, errors.New("No access token, please use /jira connect")
	}

	oauth1Config, err := jsi.GetOAuth1Config(a)
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

func (jsi *jiraServerInstance) GetOAuth1Config(a *Action) (*oauth1.Config, error) {
	rsaKey, err := a.Plugin.EnsureRSAKey()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create an OAuth1 config")
	}

	return &oauth1.Config{
		ConsumerKey:    jsi.MattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    a.Plugin.GetPluginURL() + "/" + routeOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: jsi.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    jsi.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  jsi.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
	}, nil
}

func publicKeyString(p *Plugin) ([]byte, error) {
	rsaKey, err := p.EnsureRSAKey()
	if err != nil {
		return nil, err
	}

	b, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to encode public key")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}
