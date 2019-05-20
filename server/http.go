// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/plugin"
)

const (
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIUserInfo               = "/api/v2/userinfo"
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

var httpRequireJiraClient = []ActionFunc{
	RequireCommandMattermostUserId,
	RequireJiraClient,
}

var httpRouter = ActionRouter{
	DefaultRouteHandler: func(a *Action) error {
		return a.RespondError(http.StatusNotFound, nil, "not found")
	},
	Log: []ActionFunc{
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
		routeAPICreateIssue: {
			Filters: []ActionFunc{RequireHTTPPost, RequireHTTPMattermostUserId, RequireJiraClient},
			Handler: httpAPICreateIssue,
		},
		routeAPIGetCreateIssueMetadata: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPMattermostUserId, RequireJiraClient},
			Handler: httpAPIGetCreateIssueMetadata,
		},
		routeAPIUserInfo: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPMattermostUserId, RequireJiraUser},
			Handler: httpAPIGetUserInfo,
		},
		// Atlassian Connect application
		routeACInstalled: {
			Filters: []ActionFunc{RequireHTTPPost},
			Handler: httpACInstalled,
		},
		routeACJSON: {
			Filters: []ActionFunc{RequireHTTPGet},
			Handler: httpACJSON,
		},

		// User connect and disconnect URLs
		routeUserConnect: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPMattermostUserId, RequireInstance},
			Handler: httpUserConnect,
		},
		routeUserDisconnect: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPMattermostUserId, RequireInstance, RequireJiraUser},
			Handler: httpUserDisconnect,
		},

		// Atlassian Connect user mapping
		routeACUserRedirectWithToken: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPCloudJWT},
			Handler: httpACUserRedirect,
		},
		routeACUserConfirm: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPCloudJWT},
			Handler: httpACUserInteractive,
		},
		routeACUserConnected: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPCloudJWT},
			Handler: httpACUserInteractive,
		},
		routeACUserDisconnected: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPCloudJWT},
			Handler: httpACUserInteractive,
		},

		// Oauth1 (Jira Server) user mapping
		routeOAuth1Complete: {
			Filters: []ActionFunc{RequireHTTPGet, RequireHTTPMattermostUserId, RequireServerInstance, RequireMattermostUser},
			Handler: httpOAuth1Complete,
		},

		// incoming webhooks
		routeIncomingWebhook: {
			Filters: []ActionFunc{RequireHTTPPost},
			Handler: httpWebhook,
		},
		routeIncomingIssueEvent: {
			Filters: []ActionFunc{RequireHTTPPost},
			Handler: httpWebhook,
		},
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

func (p *Plugin) loadTemplates(dir string) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		template, err := template.ParseFiles(path)
		if err != nil {
			p.errorf("OnActivate: failed to parse template %s: %v", path, err)
			return nil
		}
		key := path[len(dir):]
		templates[key] = template
		p.debugf("loaded template %s", key)
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "OnActivate: failed to load templates")
	}
	return templates, nil
}
