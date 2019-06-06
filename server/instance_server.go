// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/x509"
	"encoding/pem"

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

func NewJIRAServerInstance(jiraURL, mattermostKey string) *jiraServerInstance {
	return &jiraServerInstance{
		JIRAInstance:  newJIRAInstance(JIRATypeServer, jiraURL),
		MattermostKey: mattermostKey,
		JIRAServerURL: jiraURL,
	}
}

func (jsi jiraServerInstance) GetURL() string {
	return jsi.JIRAServerURL
}

func (jsi jiraServerInstance) GetMattermostKey() string {
	return jsi.MattermostKey
}

func (jsi jiraServerInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": jsi.MattermostKey,
	}
}

func (jsi jiraServerInstance) GetUserConnectURL(conf Config, secretsStore SecretsStore,
	mattermostUserId string) (returnURL string, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
		}
	}()

	oauth1Config, err := jsi.GetOAuth1Config(conf, secretsStore)
	if err != nil {
		return "", err
	}

	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", err
	}

	err = secretsStore.StoreOauth1aTemporaryCredentials(mattermostUserId,
		&OAuth1aTemporaryCredentials{Token: token, Secret: secret})
	if err != nil {
		return "", err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", err
	}

	return authURL.String(), nil
}

func (jsi jiraServerInstance) GetJIRAClient(conf Config, secretsStore SecretsStore,
	jiraUser *JIRAUser) (returnClient *jira.Client, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessagef(returnErr,
				"failed to get a Jira client for %q", jiraUser.Name)
		}
	}()

	if jiraUser.Oauth1AccessToken == "" || jiraUser.Oauth1AccessSecret == "" {
		return nil, errors.New("No access token, please use /jira connect")
	}

	oauth1Config, err := jsi.GetOAuth1Config(conf, secretsStore)
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

func (jsi *jiraServerInstance) GetOAuth1Config(conf Config, secretsStore SecretsStore) (*oauth1.Config, error) {
	rsaKey, err := secretsStore.EnsureRSAKey()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create an OAuth1 config")
	}

	return &oauth1.Config{
		ConsumerKey:    jsi.MattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    conf.PluginURL + "/" + routeOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: jsi.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    jsi.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  jsi.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
	}, nil
}

func publicKeyString(secretsStore SecretsStore) ([]byte, error) {
	rsaKey, err := secretsStore.EnsureRSAKey()
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
