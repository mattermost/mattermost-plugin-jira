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
	case "/test":
		return p.serveTest(w, r)
	case "/oauth/connect":
		return p.serveOAuth2Connect(w, r)
	case "/oauth/complete":
		return p.serveOAuth2Complete(w, r)
	case "/webhook",
		"/issue_event":
		return p.handleWebhook(w, r)
	case "/atlassian-connect.json":
		return p.serveAtlassianConnect(w, r)
	case "/installed":
		return p.serveInstalled(w, r)
	case "/uninstalled":
		return p.serveUninstalled(w, r)
	case "/create-issue":
		return p.serveCreateIssue(w, r)
	case "/create-issue-metadata":
		return p.serveCreateIssueMetadata(w, r)
	case "/api/v1/connected":
		return p.serveGetConnected(w, r)
	}

	return http.StatusNotFound, fmt.Errorf("Not found")
}
