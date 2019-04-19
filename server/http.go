// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-server/plugin"
)

const (
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIUserInfo               = "/api/v2/userinfo"
	routeACInstalled               = "/ac/installed"
	routeACJSON                    = "/ac/atlassian-connect.json"
	routeACUninstalled             = "/ac/uninstalled"
	routeACUserConfig              = "/ac/user-config"
	routeACUserConfigSubmit        = "/ac/user-config-submit"
	routeIncomingIssueEvent        = "/issue_event"
	routeIncomingWebhook           = "/webhook"
	routeOAuth1Complete            = "/oauth1/complete"
	routeOAuth1Connect             = "/oauth1/connect"
	routeOAuth1PublicKey           = "/oauth1/public-key"
	routeUserConnect               = "/user/connect"
	routeUserDisconnect            = "/user/disconnect"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfig()
	if config.UserName == "" {
		http.Error(w, "JIRA plugin not configured correctly; must provide UserName", http.StatusForbidden)
		return
	}

	status, err := handleHTTPRequest(p, w, r)
	if err != nil {
		p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
		http.Error(w, err.Error(), status)
		return
	}
	p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
}

func handleHTTPRequest(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	switch r.URL.Path {
	// Issue APIs
	case routeAPICreateIssue:
		return httpAPICreateIssue(p, w, r)
	case routeAPIGetCreateIssueMetadata:
		return httpAPIGetCreateIssueMetadata(p, w, r)

	// User APIs
	case routeAPIUserInfo:
		return httpAPIGetUserInfo(p, w, r)

	// Atlassian Connect application
	case routeACInstalled:
		return httpACInstalled(p, w, r)
	case routeACJSON:
		return httpACJSON(p, w, r)
	case routeACUninstalled:
		return httpACUninstalled(p, w, r)

	// Atlassian Connect user mapping
	case routeACUserConfig:
		return httpACUserConfig(p, w, r)
	case routeACUserConfigSubmit:
		return httpACUserConfigSubmit(p, w, r)

	// Incoming webhook
	case routeIncomingWebhook, routeIncomingIssueEvent:
		return httpWebhook(p, w, r)

	// Oauth1 (JIRA Server)
	case routeOAuth1Complete:
		return httpOAuth1Complete(p, w, r)
	case routeOAuth1Connect:
		return httpOAuth1Connect(p, w, r)
	case routeOAuth1PublicKey:
		return httpOAuth1PublicKey(p, w, r)

	// User connect/disconnect links
	case routeUserConnect:
		return httpUserConnect(p, w, r)
	case routeUserDisconnect:
		return httpUserDisconnect(p, w, r)
	}

	return http.StatusNotFound, fmt.Errorf("Not found")
}
