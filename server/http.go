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
	routeAPISubscriptionsChannel   = "/api/v2/subscriptions/channel/*"
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
	routeOAuth1PublicKey           = "/oauth1/public_key.html" // TODO remove, debugging?
	routeUserConnect               = "/user/connect"
	routeUserDisconnect            = "/user/disconnect"
)

func httpPostFilter(ff ...ActionFunc) ActionFilter {
	return append(ActionFilter{RequireHTTPPost}, ff...)
}

func httpGetFilter(ff ...ActionFunc) ActionFilter {
	return append(ActionFilter{RequireHTTPGet}, ff...)
}

var httpInstanceFilter = ActionFilter{RequireHTTPMattermostUserId, RequireInstance}
var httpJiraUserFilter = ActionFilter{RequireHTTPMattermostUserId, RequireInstance, RequireJiraUser}
var httpJiraClientFilter = append(httpJiraUserFilter, RequireJiraClient)

var httpRouter = ActionRouter{
	DefaultRouteHandler: func(a *Action) error {
		return a.RespondError(http.StatusNotFound, nil, "not found")
	},
	Log: ActionFilter{
		func(a *Action) error {
			if a.HTTPStatusCode == 0 {
				a.HTTPStatusCode = http.StatusOK
			}
			if a.Err != nil {
				a.Errorf("http: %v %s %v", a.HTTPStatusCode, a.HTTPRequest.URL.String(), a.Err)
			} else {
				a.Debugf("http: %v %s", a.HTTPStatusCode, a.HTTPRequest.URL.String())
			}
			return nil
		},
	},
	RouteHandlers: map[string]*ActionScript{
		// MM client APIs
		routeAPICreateIssue: {
			httpAPICreateIssue,
			httpPostFilter(httpJiraClientFilter...)},
		routeAPIAttachCommentToIssue: {
			httpAPIAttachCommentToIssue,
			httpPostFilter(httpJiraClientFilter...)},
		routeAPIGetCreateIssueMetadata: {
			httpAPIGetCreateIssueMetadata,
			httpGetFilter(httpJiraClientFilter...)},
		routeAPIUserInfo: {
			httpAPIGetUserInfo,
			httpGetFilter(httpJiraUserFilter...)},
		routeAPISubscribeWebhook: {
			httpSubscribeWebhook,
			httpPostFilter()},
		routeAPISubscriptionsChannel: {
			httpChannelSubscriptions,
			ActionFilter{RequireHTTPMattermostUserId}},

		// Incoming webhooks
		routeIncomingWebhook:    {httpWebhook, httpPostFilter()},
		routeIncomingIssueEvent: {httpWebhook, httpPostFilter()},

		// Atlassian Connect application
		routeACInstalled: {httpACInstalled, httpPostFilter()},
		routeACJSON:      {httpACJSON, httpGetFilter()},

		// User connect and disconnect URLs
		routeUserConnect: {
			httpUserConnect,
			httpGetFilter(RequireHTTPMattermostUserId, RequireInstance)},
		routeUserDisconnect: {
			httpUserDisconnect,
			httpGetFilter(httpJiraUserFilter...)},

		// Atlassian Connect user mapping
		routeACUserRedirectWithToken: {
			httpACUserRedirect,
			httpGetFilter(RequireHTTPCloudJWT)},
		routeACUserConfirm: {
			httpACUserInteractive,
			httpGetFilter(RequireHTTPMattermostUserId, RequireMattermostUser, RequireHTTPCloudJWT)},
		routeACUserConnected: {
			httpACUserInteractive,
			httpGetFilter(RequireHTTPMattermostUserId, RequireMattermostUser, RequireHTTPCloudJWT)},
		routeACUserDisconnected: {
			httpACUserInteractive,
			httpGetFilter(RequireHTTPMattermostUserId, RequireMattermostUser, RequireHTTPCloudJWT)},

		// Oauth1 (Jira Server) user mapping
		routeOAuth1Complete: {
			httpOAuth1Complete,
			httpGetFilter(RequireHTTPMattermostUserId, RequireServerInstance, RequireMattermostUser)},
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
