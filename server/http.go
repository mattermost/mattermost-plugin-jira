// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	goexpvar "expvar"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata-for-project"
	routeAPIGetJiraProjectMetadata = "/api/v2/get-jira-project-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
	routeAPIGetSearchEpics         = "/api/v2/get-search-epics"
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPIUserInfo               = "/api/v2/userinfo"
	routeAPISubscribeWebhook       = "/api/v2/webhook"
	routeAPISubscriptionsChannel   = "/api/v2/subscriptions/channel"
	routeAPISettingsInfo           = "/api/v2/settingsinfo"
	routeAPIStats                  = "/api/v2/stats"
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

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	status, err := handleHTTPRequest(p, w, r)
	if err != nil {
		p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
		http.Error(w, err.Error(), status)
		return
	}
	switch status {
	case http.StatusOK:
		// pass through
	case 0:
		status = http.StatusOK
	default:
		w.WriteHeader(status)
	}
	p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
}

func handleHTTPRequest(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	switch r.URL.Path {
	// Issue APIs
	case routeAPICreateIssue:
		return withInstance(p.currentInstanceStore, w, r, httpAPICreateIssue)
	case routeAPIGetCreateIssueMetadata:
		return withInstance(p.currentInstanceStore, w, r, httpAPIGetCreateIssueMetadataForProjects)
	case routeAPIGetJiraProjectMetadata:
		return withInstance(p.currentInstanceStore, w, r, httpAPIGetJiraProjectMetadata)
	case routeAPIGetSearchIssues:
		return withInstance(p.currentInstanceStore, w, r, httpAPIGetSearchIssues)
	case routeAPIGetSearchEpics:
		return withInstance(p.currentInstanceStore, w, r, httpAPIGetSearchEpics)
	case routeAPIAttachCommentToIssue:
		return withInstance(p.currentInstanceStore, w, r, httpAPIAttachCommentToIssue)

	// User APIs
	case routeAPIUserInfo:
		return httpAPIGetUserInfo(p, w, r)
	case routeAPISettingsInfo:
		return httpAPIGetSettingsInfo(p, w, r)

	// Stats
	case routeAPIStats:
		return httpAPIStats(p, w, r)

	// Atlassian Connect application
	case routeACInstalled:
		return httpACInstalled(p, w, r)
	case routeACJSON:
		return httpACJSON(p, w, r)
	case routeACUninstalled:
		return httpACUninstalled(p, w, r)

	// Atlassian Connect user mapping
	case routeACUserRedirectWithToken:
		return withCloudInstance(p, w, r, httpACUserRedirect)
	case routeACUserConfirm,
		routeACUserConnected,
		routeACUserDisconnected:
		return withCloudInstance(p, w, r, httpACUserInteractive)

	// Incoming webhook
	case routeIncomingWebhook, routeIncomingIssueEvent:
		return httpWebhook(p, w, r)

	// Oauth1 (Jira Server)
	case routeOAuth1Complete:
		return withServerInstance(p, w, r, httpOAuth1aComplete)
	case routeUserDisconnect:
		return withServerInstance(p, w, r, httpOAuth1aDisconnect)
	case routeOAuth1PublicKey:
		return httpOAuth1aPublicKey(p, w, r)

	// User connect/disconnect links
	case routeUserConnect:
		return withInstance(p.currentInstanceStore, w, r, httpUserConnect)
	// Firehose webhook setup for channel subscriptions
	case routeAPISubscribeWebhook:
		return httpSubscribeWebhook(p, w, r)

	// expvar
	case "/debug/vars":
		goexpvar.Handler().ServeHTTP(w, r)
		return 0, nil
	}

	if strings.HasPrefix(r.URL.Path, routeAPISubscriptionsChannel) {
		return httpChannelSubscriptions(p, w, r)
	}
	return http.StatusNotFound, errors.New("not found")
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

func (p *Plugin) respondWithTemplate(w http.ResponseWriter, r *http.Request, contentType string, values interface{}) (int, error) {
	w.Header().Set("Content-Type", contentType)
	t := p.templates[r.URL.Path]
	if t == nil {
		return http.StatusInternalServerError,
			errors.New("no template found for " + r.URL.Path)
	}
	err := t.Execute(w, values)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

func (p *Plugin) respondSpecialTemplate(w http.ResponseWriter, key string, status int, contentType string, values interface{}) (int, error) {
	w.Header().Set("Content-Type", contentType)
	t := p.templates[key]
	if t == nil {
		return http.StatusInternalServerError,
			errors.New("no template found for " + key)
	}
	err := t.Execute(w, values)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}
	return status, nil
}
