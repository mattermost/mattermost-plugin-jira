// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// Leaving this commented out code here as a comparison of the original function

// func newCloudInstance(p *Plugin, key types.ID, installed bool, rawASC string, asc *AtlassianSecurityContext) *cloudInstance {
// 	return &cloudInstance{
// 		InstanceCommon:              newInstanceCommon(p, CloudInstanceType, key),
// 		Installed:                   installed,
// 		RawAtlassianSecurityContext: rawASC,
// 		AtlassianSecurityContext:    asc,
// 	}
// }

func newCloudInstanceOAuth(p *Plugin, jiraURL string) *cloudInstance {
	return &cloudInstance{
		InstanceCommon:   newInstanceCommon(p, CloudInstanceType, types.ID(jiraURL)),
		MattermostKey:    p.GetPluginKey(),
		Installed:        true, // APPLINKTODO: not sure what should be done here. This is called during install, and will probably be make to be called in kv.go later
		InstalledAppLink: true,
	}
}

func (ci *cloudInstance) GetURLOAuth() string {
	return ci.InstanceID.String()
}

func (ci *cloudInstance) GetMattermostKeyOAuth() string {
	return ci.MattermostKey
}

func (ci *cloudInstance) GetDisplayDetailsOAuth() map[string]string {
	return map[string]string{
		"Jira Server Mattermost Key": ci.MattermostKey,
	}
}

func (ci *cloudInstance) GetUserConnectURLOAuth(mattermostUserID string) (returnURL string, cookie *http.Cookie, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
	}()

	oauth1Config := ci.getOAuth1Config()
	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", nil, err
	}

	err = ci.Plugin.otsStore.StoreOauth1aTemporaryCredentials(mattermostUserID,
		&OAuth1aTemporaryCredentials{Token: token, Secret: secret})
	if err != nil {
		return "", nil, err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", nil, err
	}

	return authURL.String(), nil, nil
}

func (ci *cloudInstance) getClientForConnectionOAuth(connection *Connection) (jiraClient *jira.Client, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a Jira client for "+connection.DisplayName)
	}()

	if connection.Oauth1AccessToken == "" || connection.Oauth1AccessSecret == "" {
		return nil, errors.New("no access token, please use /jira connect")
	}

	token := oauth1.NewToken(connection.Oauth1AccessToken, connection.Oauth1AccessSecret)
	conf := ci.getConfig()

	httpClient := ci.getOAuth1Config().Client(oauth1.NoContext, token)
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))
	httpClient = expvar.WrapHTTPClient(httpClient,
		conf.stats, endpointNameFromRequest)

	jiraClient, err := jira.NewClient(httpClient, ci.GetURL())
	if err != nil {
		return nil, err
	}

	return jiraClient, err
}

func (ci *cloudInstance) getOAuth1Config() *oauth1.Config {
	p := ci.Plugin
	return &oauth1.Config{
		ConsumerKey:    ci.MattermostKey,
		ConsumerSecret: "consumer_secret",
		CallbackURL:    p.GetPluginURL() + "/" + instancePath(routeOAuth1Complete, ci.InstanceID),
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: ci.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    ci.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  ci.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{
			PrivateKey: p.getConfig().rsaKey,
		},
	}
}
