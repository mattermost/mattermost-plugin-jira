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
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPIUserInfo               = "/api/v2/userinfo"
	routeAPISubscribeWebhook       = "/api/v2/webhook"
	routeAPISubscriptionsChannel   = "/api/v2/subscriptions/channel"
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
	return append(ActionFilter{RequireHTTPPost}, ff...)
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
				a.Plugin.errorf("http: %v %s %v", a.HTTPStatusCode, a.HTTPRequest.URL.String(), a.Err)
			} else {
				a.Plugin.debugf("http: %v %s", a.HTTPStatusCode, a.HTTPRequest.URL.String())
			}
			return nil
		},
	},
	RouteHandlers: map[string]*ActionScript{
		// MM client APIs
		routeAPICreateIssue:            {Filter: httpPostFilter(httpJiraClientFilter...), Handler: httpAPICreateIssue},
		routeAPIAttachCommentToIssue:   {Filter: httpPostFilter(httpJiraClientFilter...), Handler: httpAPIAttachCommentToIssue},
		routeAPIGetCreateIssueMetadata: {Filter: httpGetFilter(httpJiraClientFilter...), Handler: httpAPIGetCreateIssueMetadata},
		routeAPIUserInfo:               {Filter: httpGetFilter(httpJiraUserFilter...), Handler: httpAPIGetUserInfo},
		routeAPISubscribeWebhook:       {Filter: httpPostFilter(), Handler: httpSubscribeWebhook},
		routeAPISubscriptionsChannel:   {Filter: ActionFilter{RequireHTTPMattermostUserId}, Handler: httpChannelSubscriptions},

		// Atlassian Connect application
		routeACInstalled: {Filter: httpPostFilter(), Handler: httpACInstalled},
		routeACJSON:      {Filter: httpGetFilter(), Handler: httpACJSON},

		// User connect and disconnect URLs
		routeUserConnect:    {Filter: httpGetFilter(RequireHTTPMattermostUserId, RequireInstance), Handler: httpUserConnect},
		routeUserDisconnect: {Filter: httpGetFilter(httpJiraUserFilter...), Handler: httpUserDisconnect},

		// Atlassian Connect user mapping
		routeACUserRedirectWithToken: {Filter: httpGetFilter(RequireHTTPCloudJWT), Handler: httpACUserRedirect},
		routeACUserConfirm:           {Filter: httpGetFilter(RequireHTTPCloudJWT), Handler: httpACUserInteractive},
		routeACUserConnected:         {Filter: httpGetFilter(RequireHTTPCloudJWT), Handler: httpACUserInteractive},
		routeACUserDisconnected:      {Filter: httpGetFilter(RequireHTTPCloudJWT), Handler: httpACUserInteractive},

		// Oauth1 (Jira Server) user mapping
		routeOAuth1Complete: {Filter: httpGetFilter(RequireHTTPMattermostUserId, RequireServerInstance, RequireMattermostUser), Handler: httpOAuth1Complete},

		// incoming webhooks
		routeIncomingWebhook:    {Filter: httpPostFilter(), Handler: httpWebhook},
		routeIncomingIssueEvent: {Filter: httpPostFilter(), Handler: httpWebhook},
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
