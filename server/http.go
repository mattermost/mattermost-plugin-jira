// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"

	"github.com/mattermost/mattermost-server/plugin"
)

const (
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPIUserInfo               = "/api/v2/userinfo"
	routeAPISubscribeWebhook       = "/api/v2/webhook"
	routeAPISubscriptionsChannel   = "/api/v2/subscriptions/channel/" // trailing '/' on purpose
	routeACInstalled               = "/ac/installed"
	routeACJSON                    = "/ac/atlassian-connect.json"
	routeACUninstalled             = "/ac/uninstalled"
	routeACUserRedirectWithToken   = "/ac/user_redirect.html"
	routeACUserConfirm             = "/ac/user_confirm.html"
	routeACUserConnected           = "/ac/user_connected.html"
	routeACUserDisconnected        = "/ac/user_disconnected.html"
	routeIncomingIssueEvent        = "/issue_event"
	routeIncomingWebhook           = "/webhook"
	routeOAuth1Complete            = "/oauth1/complete.html"
	routeOAuth1PublicKey           = "/oauth1/public_key.html"
	routeUserConnect               = "/user/connect"
	routeUserDisconnect            = "/user/disconnect"
)

var httpRouter = ActionRouter{
	DefaultRouteHandler: func(a *Action) error {
		return a.RespondError(http.StatusNotFound, nil, "not found")
	},
	Log: func(a *Action) error {
		if a.HTTPStatusCode == 0 {
			a.HTTPStatusCode = http.StatusOK
		}
		if a.LogErr != nil {
			a.Infof("http: %v %s %v", a.HTTPStatusCode, a.HTTPRequest.URL.String(), a.LogErr)
		} else {
			a.Debugf("http: %v %s", a.HTTPStatusCode, a.HTTPRequest.URL.String())
		}
		return nil
	},
	RouteHandlers: map[string]ActionScript{
		// APIs
		routeAPICreateIssue:            httpAPICreateIssue,
		routeAPIAttachCommentToIssue:   httpAPIAttachCommentToIssue,
		routeAPIGetSearchIssues:        httpAPIGetSearchIssues,
		routeAPIGetCreateIssueMetadata: httpAPIGetCreateIssueMetadata,
		routeAPIUserInfo:               httpAPIGetUserInfo,
		routeAPISubscribeWebhook:       httpSubscribeWebhook,

		// httpChannelSubscriptions already ends in a '/', so adding "*" will
		// pass all sub-paths up to the handler
		routeAPISubscriptionsChannel + "*": httpChannelSubscriptions,

		// Incoming webhooks
		routeIncomingWebhook:    httpWebhook,
		routeIncomingIssueEvent: httpWebhook,

		// Atlassian Connect application
		routeACInstalled: httpACInstalled,
		routeACJSON:      httpACJSON,

		// User connect and disconnect URLs
		routeUserConnect:    httpUserConnect,
		routeUserDisconnect: httpUserDisconnect,

		// Atlassian Connect user mapping
		routeACUserRedirectWithToken: httpACUserRedirect,
		routeACUserConfirm:           httpACUserConfirm,
		routeACUserConnected:         httpACUserConnected,
		routeACUserDisconnected:      httpACUserDisconnected,

		// Oauth1 (Jira Server) user mapping
		routeOAuth1Complete: httpOAuth1Complete,
	},
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	action := NewAction(p, c)
	action.HTTPRequest = r
	action.HTTPResponseWriter = w
	if action.PluginConfig.UserName == "" {
		http.Error(w, "Jira plugin not configured correctly; must provide UserName", http.StatusForbidden)
		return
	}

	httpRouter.Run(r.URL.Path, action)
}
