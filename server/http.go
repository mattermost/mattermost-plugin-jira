// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()
	if config.UserName == "" {
		http.Error(w, "JIRA plugin not configured correctly; must provide UserName", http.StatusForbidden)
		return
	}

	status, err := p.handleHTTPRequest(w, r)
	if err != nil {
		p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
		http.Error(w, err.Error(), status)
	}
	p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
}

func (p *Plugin) handleHTTPRequest(w http.ResponseWriter, r *http.Request) (int, error) {
	switch r.URL.Path {
	// Atlassian connect and its "lifecycle events"
	case "/atlassian-connect.json":
		return p.handleHTTPAtlassianConnect(w, r)
	case "/installed":
		return p.handleHTTPInstalled(w, r)
	case "/uninstalled":
		return p.handleHTTPUninstalled(w, r)

	// OAuth end-points
	case "/oauth/connect":
		return p.handleHTTPOAuth2Connect(w, r)
	case "/oauth/complete":
		return p.handleHTTPOAuth2Complete(w, r)

	case "/webhook",
		"/issue_event":
		return p.handleHTTPWebhook(w, r)

	// User mapping page
	case "/user-connect":
		return p.handleHTTPUserConnect(w, r)
	case "/config":
		return p.handleHTTPUserConfig(w, r)
	case "/api/v1/userinfo":
		return p.handleHTTPGetUserInfo(w, r)

	case "/create-issue":
		return p.handleHTTPCreateIssue(w, r)
	case "/test":
		return p.serveTest(w, r)
	case "/create-issue-metadata":
		return p.serveCreateIssueMetadata(w, r)
	}

	return http.StatusNotFound, fmt.Errorf("Not found")
}
