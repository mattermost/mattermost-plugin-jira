// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type serverInstance struct {
	*InstanceCommon

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	DeprecatedJIRAServerURL string `json:"JIRAServerURL"`
}

var _ Instance = (*serverInstance)(nil)

func (p *Plugin) installServerInstance(rawURL string) (string, *serverInstance, error) {
	jiraURL, err := utils.CheckJiraURL(p.GetSiteURL(), rawURL, false)
	if err != nil {
		return "", nil, err
	}
	if utils.IsJiraCloudURL(jiraURL) {
		return "", nil, errors.Errorf("`%s` is not a Jira server URL, it refers to Jira Cloud", jiraURL)
	}

	instance := &serverInstance{
		InstanceCommon: newInstanceCommon(p, ServerInstanceType, types.ID(jiraURL)),
		MattermostKey:  p.GetPluginKey(),
	}

	err = p.InstallInstance(instance)
	if err != nil {
		return "", nil, err
	}

	return jiraURL, instance, err
}

func (si *serverInstance) GetURL() string {
	return si.InstanceID.String()
}

func (si *serverInstance) GetJiraBaseURL() string {
	return si.GetURL()
}

func (si *serverInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/plugins/servlet/applinks/listApplicationLinks", si.GetURL())
}

func (si *serverInstance) GetManageWebhooksURL() string {
	return fmt.Sprintf("%s/plugins/servlet/webhooks", si.GetURL())
}

func (si *serverInstance) GetMattermostKey() string {
	return si.MattermostKey
}

func (si *serverInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Jira Server Mattermost Key": si.MattermostKey,
	}
}

func (si *serverInstance) GetUserConnectURL(mattermostUserID string) (returnURL string, cookie *http.Cookie, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
	}()

	oauth1Config := si.getOAuth1Config()
	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", nil, err
	}

	err = si.Plugin.otsStore.StoreOauth1aTemporaryCredentials(mattermostUserID,
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

func (si *serverInstance) GetClient(connection *Connection) (client Client, returnErr error) {
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
	conf := si.getConfig()

	httpClient := si.getOAuth1Config().Client(oauth1.NoContext, token)
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))

	jiraClient, err := jira.NewClient(httpClient, si.GetURL())
	if err != nil {
		return nil, err
	}

	return newServerClient(jiraClient), nil
}

func (si *serverInstance) getOAuth1Config() *oauth1.Config {
	p := si.Plugin
	return &oauth1.Config{
		ConsumerKey:    si.MattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    p.GetPluginURL() + "/" + instancePath(routeOAuth1Complete, si.InstanceID),
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: si.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    si.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  si.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{
			PrivateKey: p.getConfig().rsaKey,
		},
	}
}
