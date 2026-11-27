// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/httpservice"
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

const jiraRequestTimeout = 30 * time.Second

// errJiraServerRequestFailed is shown to users in place of transport and
// upstream errors, which are logged instead.
var errJiraServerRequestFailed = errors.New("we couldn't complete the request to the Jira server. Please try again later or contact your system administrator")

type restrictedTransport struct {
	insecure bool
	rt       http.RoundTripper
}

// getRestrictedTransport returns a transport that blocks internal addresses,
// on every redirect hop and whatever the host resolves to at request time,
// unless they are in AllowedUntrustedInternalConnections. The allow list is
// read on every dial; EnableInsecureOutgoingConnections is fixed per
// transport, so the transport is rebuilt when it changes.
func (p *Plugin) getRestrictedTransport() http.RoundTripper {
	insecure := model.SafeDereference(p.API.GetConfig().ServiceSettings.EnableInsecureOutgoingConnections)
	if t := p.restrictedTransport.Load(); t != nil && t.insecure == insecure {
		return t.rt
	}
	// The wrapped transport is used directly so the plugin's User-Agent is kept.
	t := &restrictedTransport{insecure: insecure, rt: httpservice.MakeHTTPServicePlugin(p.API).MakeTransport(false).Transport}
	p.restrictedTransport.Store(t)
	return t.rt
}

// restrictedHTTPClient returns a client for requests to Jira.
// A zero timeout means no timeout.
func (p *Plugin) restrictedHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Transport: p.getRestrictedTransport(), Timeout: timeout}
}

// instanceHTTPClient is restrictedHTTPClient for an installed instance; it
// tells the admins when the instance's requests are being blocked.
func (p *Plugin) instanceHTTPClient(instance Instance, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &blockedNoticeTransport{
			rt:        p.getRestrictedTransport(),
			onBlocked: func() { p.notifyAdminsIfInstanceBlocked(instance) },
		},
		Timeout: timeout,
	}
}

type blockedNoticeTransport struct {
	rt        http.RoundTripper
	onBlocked func()
}

func (t *blockedNoticeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.rt.RoundTrip(req)
	// httpservice adds detail to ErrAddressForbidden without wrapping it.
	if err != nil && strings.Contains(err.Error(), httpservice.ErrAddressForbidden.Error()) {
		t.onBlocked()
	}
	return resp, err
}

func (p *Plugin) checkJiraURL(rawURL string) (string, error) {
	client := p.restrictedHTTPClient(jiraRequestTimeout)
	jiraURL, err := utils.CheckJiraURL(client, p.GetSiteURL(), rawURL, false)
	var checkErr *utils.JiraURLCheckError
	if errors.As(err, &checkErr) {
		p.client.Log.Warn("Failed to validate the connection to the Jira server", "error", checkErr.Cause.Error())
	}
	return jiraURL, err
}

func (p *Plugin) installServerInstance(rawURL string) (string, *serverInstance, error) {
	jiraURL, err := utils.NormalizeJiraURL(rawURL)
	if err != nil {
		return "", nil, err
	}
	if utils.IsJiraCloudURL(jiraURL) {
		return "", nil, errors.Errorf("`%s` is not a Jira server URL, it refers to Jira Cloud", jiraURL)
	}
	jiraURL, err = p.checkJiraURL(jiraURL)
	if err != nil {
		return "", nil, err
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
	return normalizeJiraBaseURL(si.GetURL())
}

func (si *serverInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/plugins/servlet/applinks/listApplicationLinks", si.GetJiraBaseURL())
}

func (si *serverInstance) GetManageWebhooksURL() string {
	return fmt.Sprintf("%s/plugins/servlet/webhooks", si.GetJiraBaseURL())
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
		si.Plugin.client.Log.Warn("Failed to obtain an OAuth1 request token from Jira", "error", err.Error())
		return "", nil, errJiraServerRequestFailed
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
		returnErr = errors.WithMessage(returnErr, fmt.Sprintf("failed to get a Jira client for %s", connection.Name))
	}()

	if connection.Oauth1AccessToken == "" || connection.Oauth1AccessSecret == "" {
		return nil, errors.New("no access token, please use /jira connect")
	}

	token := oauth1.NewToken(connection.Oauth1AccessToken, connection.Oauth1AccessSecret)
	conf := si.getConfig()

	ctx := context.WithValue(context.Background(), oauth1.HTTPClient, si.Plugin.instanceHTTPClient(si, 0))
	httpClient := si.getOAuth1Config().Client(ctx, token)
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
		HTTPClient: p.instanceHTTPClient(si, jiraRequestTimeout),
	}
}
