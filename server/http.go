// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/plugin"
)

const (
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPIUserInfo               = "/api/v2/userinfo"
	routeAPISettingsInfo           = "/api/v2/settingsinfo"
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
	routePluginIcon                = "/static/v2.0/icon.png"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfig()
	if config.UserName == "" {
		http.Error(w, "Jira plugin not configured correctly; must provide UserName", http.StatusForbidden)
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
		return withInstance(p, w, r, httpAPICreateIssue)
	case routeAPIGetCreateIssueMetadata:
		return withInstance(p, w, r, httpAPIGetCreateIssueMetadata)
	case routeAPIGetSearchIssues:
		return withInstance(p, w, r, httpAPIGetSearchIssues)
	case routeAPIAttachCommentToIssue:
		return withInstance(p, w, r, httpAPIAttachCommentToIssue)

	// User APIs
	case routeAPIUserInfo:
		return httpAPIGetUserInfo(p, w, r)
	case routeAPISettingsInfo:
		return httpAPIGetSettingsInfo(p, w, r)

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
		return withServerInstance(p, w, r, httpOAuth1Complete)
	case routeOAuth1PublicKey:
		return httpOAuth1PublicKey(p, w, r)

	// User connect/disconnect links
	case routeUserConnect:
		return withInstance(p, w, r, httpUserConnect)
	case routeUserDisconnect:
		return withInstance(p, w, r, httpUserDisconnect)

	case routePluginIcon:
		return httpPublicPluginIcon(p, w, r)
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

var iconData []byte

func init() {
	const icon = `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`
	iconData = make([]byte, base64.StdEncoding.DecodedLen(len(icon)))
	_, _ = base64.StdEncoding.Decode(iconData, []byte(icon))
}

func httpPublicPluginIcon(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	w.Header().Set("Content-Type", "image/png")
	_, err := w.Write(iconData)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
