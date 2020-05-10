// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type serverInstance struct {
	*InstanceCommon

	JIRAServerURL string

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	// oauth1Config *oauth1.Config
}

var _ Instance = (*serverInstance)(nil)

func newServerInstance(p *Plugin, jiraURL string) *serverInstance {
	return &serverInstance{
		InstanceCommon: newInstanceCommon(p, ServerInstanceType, types.ID(jiraURL)),
		MattermostKey:  p.GetPluginKey(),
		JIRAServerURL:  jiraURL,
	}
}

func (si *serverInstance) GetURL() string {
	return si.JIRAServerURL
}

func (si *serverInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/plugins/servlet/applinks/listApplicationLinks", si.GetURL())
}

func (si *serverInstance) GetMattermostKey() string {
	return si.MattermostKey
}

func (si *serverInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Jira Server Mattermost Key": si.MattermostKey,
	}
}

func (si *serverInstance) GetUserConnectURL(mattermostUserId string) (returnURL string, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
	}()

	oauth1Config, err := si.getOAuth1Config()
	if err != nil {
		return "", err
	}

	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", err
	}

	err = si.Plugin.otsStore.StoreOauth1aTemporaryCredentials(mattermostUserId,
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

func (si *serverInstance) GetClient(connection *Connection) (client Client, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a Jira client for "+connection.DisplayName)
	}()

	if connection.Oauth1AccessToken == "" || connection.Oauth1AccessSecret == "" {
		return nil, errors.New("No access token, please use /jira connect")
	}

	oauth1Config, err := si.getOAuth1Config()
	if err != nil {
		return nil, err
	}

	token := oauth1.NewToken(connection.Oauth1AccessToken, connection.Oauth1AccessSecret)
	conf := si.getConfig()

	httpClient := oauth1Config.Client(oauth1.NoContext, token)
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))
	httpClient = expvar.WrapHTTPClient(httpClient,
		conf.stats, endpointNameFromRequest)

	jiraClient, err := jira.NewClient(httpClient, si.GetURL())
	if err != nil {
		return nil, err
	}

	return newServerClient(jiraClient), nil
}

func (si *serverInstance) getOAuth1Config() (returnConfig *oauth1.Config, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to create an OAuth1 config")
	}()

	rsaKey, err := si.Plugin.secretsStore.EnsureRSAKey()
	if err != nil {
		return nil, err
	}

	return &oauth1.Config{
		ConsumerKey:    si.MattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    si.Plugin.GetPluginURL() + "/" + routeOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: si.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    si.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  si.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
	}, nil
}
