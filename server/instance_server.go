// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"io/ioutil"
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

func NewJIRAServerInstance(p *Plugin, jiraURL string) Instance {
	return &jiraServerInstance{
		JIRAInstance:  NewJIRAInstance(p, JIRATypeServer, jiraURL),
		MattermostKey: p.GetPluginKey(),
		JIRAServerURL: jiraURL,
	}
}

func (jsi jiraServerInstance) GetURL() string {
	return jsi.JIRAServerURL
}

type withServerInstanceFunc func(jsi *jiraServerInstance, w http.ResponseWriter, r *http.Request) (int, error)

func withServerInstance(p *Plugin, w http.ResponseWriter, r *http.Request, f withServerInstanceFunc) (int, error) {
	return withInstance(p.currentInstanceStore, w, r, func(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
		jsi, ok := ji.(*jiraServerInstance)
		if !ok {
			return http.StatusBadRequest, errors.New("Must be a Jira Server instance, is " + ji.GetType())
		}
		return f(jsi, w, r)
	})
}

func (jsi jiraServerInstance) GetMattermostKey() string {
	return jsi.MattermostKey
}

func (jsi jiraServerInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": jsi.MattermostKey,
	}
}

func (jsi jiraServerInstance) GetUserConnectURL(mattermostUserId string) (returnURL string, returnErr error) {
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

	err = jsi.Plugin.otsStore.StoreOauth1aTemporaryCredentials(mattermostUserId,
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

func (jsi jiraServerInstance) GetJIRAClient(jiraUser JIRAUser) (returnClient *jira.Client, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get a Jira client for "+jiraUser.DisplayName)
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
	httpClient = jsi.GetPlugin().limitResponseClient(httpClient)
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

	rsaKey, err := jsi.secretsStore.EnsureRSAKey()
	if err != nil {
		return nil, err
	}

	jsi.lock.Lock()
	defer jsi.lock.Unlock()
	jsi.oauth1Config = &oauth1.Config{
		ConsumerKey:    jsi.MattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    jsi.GetPluginURL() + "/" + routeOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: jsi.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    jsi.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  jsi.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: rsaKey},
	}

	return jsi.oauth1Config, nil
}

func (jsi jiraServerInstance) GetUserGroups(jiraUser JIRAUser) ([]*jira.UserGroup, error) {
	jiraClient, err := jsi.GetJIRAClient(jiraUser)
	if err != nil {
		return nil, errors.Wrap(err, "could not get jira client")
	}

	req, err := jiraClient.NewRequest("GET", "rest/api/2/myself?expand=groups", nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request")
	}

	var groupsStruct struct {
		Groups struct {
			Items []*jira.UserGroup
		}
	}
	resp, err := jiraClient.Do(req, &groupsStruct)
	if err != nil {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, errors.Wrap(err, "error in request to get user groups, body:"+string(body))
	}

	return groupsStruct.Groups.Items, nil
}
