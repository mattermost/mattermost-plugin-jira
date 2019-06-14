// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package instance

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"
)

type jiraServerInstance struct {
	*instance

	// The JSON name is v2.0 compatible
	ServerURL string `json:"JIRAServerURL,omitempty"`

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	oauth1Config *oauth1.Config
}

var _ Instance = (*jiraServerInstance)(nil)

func NewServerInstance(jiraURL, mattermostKey string) *jiraServerInstance {
	return &jiraServerInstance{
		instance:      newInstance(InstanceTypeServer, jiraURL),
		MattermostKey: mattermostKey,
		ServerURL:     jiraURL,
	}
}

func (serverInstance jiraServerInstance) GetURL() string {
	return serverInstance.ServerURL
}

func (serverInstance jiraServerInstance) GetMattermostKey() string {
	return serverInstance.MattermostKey
}

func (serverInstance jiraServerInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": serverInstance.MattermostKey,
	}
}

func (serverInstance jiraServerInstance) GetUserConnectURL(conf Config, secretsStore SecretStore,
	mattermostUserId string) (returnURL string, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
		}
	}()

	oauth1Config, err := serverInstance.GetOAuth1Config(conf, secretsStore)
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

func (serverInstance jiraServerInstance) GetClient(conf Config, secretsStore SecretStore,
	jiraUser *JiraUser) (returnClient *jira.Client, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessagef(returnErr,
				"failed to get a Jira client for %q", jiraUser.Name)
		}
	}()

	if jiraUser.Oauth1AccessToken == "" || jiraUser.Oauth1AccessSecret == "" {
		return nil, errors.New("No access token, please use /jira connect")
	}

	oauth1Config, err := serverInstance.GetOAuth1Config(conf, secretsStore)
	if err != nil {
		return nil, err
	}

	token := oauth1.NewToken(jiraUser.Oauth1AccessToken, jiraUser.Oauth1AccessSecret)
	httpClient := oauth1Config.Client(oauth1.NoContext, token)
	jiraClient, err := jira.NewClient(httpClient, serverInstance.GetURL())
	if err != nil {
		return nil, err
	}

	return jiraClient, nil
}

func (serverInstance *jiraServerInstance) GetOAuth1Config(conf Config, secretsStore SecretStore) (*oauth1.Config, error) {
	rsaKey, err := secretsStore.EnsureRSAKey()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create an OAuth1 config")
	}

	return &oauth1.Config{
		ConsumerKey:    serverInstance.MattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    conf.PluginURL + "/" + routeOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: serverInstance.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    serverInstance.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  serverInstance.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
	}, nil
}

func publicKeyString(secretsStore SecretStore) ([]byte, error) {
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
