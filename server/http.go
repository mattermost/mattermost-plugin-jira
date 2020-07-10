// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	goexpvar "expvar"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-workflow-client/workflowclient"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata-for-project"
	routeAPIGetJiraProjectMetadata = "/api/v2/get-jira-project-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
	routeAPIGetAutoCompleteFields  = "/api/v2/get-search-autocomplete-fields"
	routeAPIGetSearchUsers         = "/api/v2/get-search-users"
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPIUserInfo               = "/api/v2/userinfo"
	routeAPISubscribeWebhook       = "/api/v2/webhook"
	routeAPISubscriptionsChannel   = "/api/v2/subscriptions/channel"
	routeAPISettingsInfo           = "/api/v2/settingsinfo"
	routeAPIStats                  = "/api/v2/stats"
	routeIssueTransition           = "/api/v2/transition"
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
	routeUserStart                 = "/user/start"
	routeUserConnect               = "/user/connect"
	routeUserDisconnect            = "/user/disconnect"
	routeWorkflowRegister          = "/workflow/meta"
	routeWorkflowTriggerSetup      = "/workflow/trigger_setup"
	routeWorkflowCreateIssue       = "/workflow/create_issue"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	status, err := p.serveHTTP(c, w, r)
	if err != nil {
		p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
	}
	p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
}

func (p *Plugin) serveHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) (int, error) {
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
	case routeAPIGetAutoCompleteFields:
		return withInstance(p.currentInstanceStore, w, r, httpAPIGetAutoCompleteFields)
	case routeAPIGetSearchUsers:
		return withInstance(p.currentInstanceStore, w, r, httpAPIGetSearchUsers)
	case routeAPIAttachCommentToIssue:
		return withInstance(p.currentInstanceStore, w, r, httpAPIAttachCommentToIssue)
	case routeIssueTransition:
		return withInstance(p.currentInstanceStore, w, r, httpAPITransitionIssue)

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
	case routeUserStart:
		return withInstance(p.currentInstanceStore, w, r, httpUserStart)
	// Firehose webhook setup for channel subscriptions
	case routeAPISubscribeWebhook:
		return httpSubscribeWebhook(p, w, r)

	// expvar
	case "/debug/vars":
		goexpvar.Handler().ServeHTTP(w, r)
		return 0, nil

	// Workflow
	case routeWorkflowRegister:
		{
			if c.SourcePluginId != "" {
				return httpWorkflowRegister(p, w, r)
			}
		}
	case routeWorkflowTriggerSetup:
		{
			if c.SourcePluginId != "" {
				return httpWorkflowTriggerSetup(p, w, r)
			}
		}
	case routeWorkflowCreateIssue:
		{
			if c.SourcePluginId != "" {
				return withInstance(p.currentInstanceStore, w, r, httpWorkflowCreateIssue)
			}
		}
	}

	if strings.HasPrefix(r.URL.Path, routeAPISubscriptionsChannel) {
		return httpChannelSubscriptions(p, w, r)
	}

	return respondErr(w, http.StatusNotFound, errors.New("not found"))
}

func httpWorkflowRegister(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	params := workflowclient.RegisterParams{
		Triggers: []workflowclient.TriggerParams{
			{
				TypeName:    "event",
				DisplayName: "Jira Event",
				Fields: []workflowclient.Field{
					{
						Name: "events",
						Type: "[]string",
					},
					{
						Name: "projects",
						Type: "[]string",
					},
					{
						Name: "issue_types",
						Type: "[]string",
					},
				},
				VarInfos: []workflowclient.VarInfo{
					{
						Name:        "Summary",
						Description: "The summary of the ticket",
					},
					{
						Name:        "Description",
						Description: "The description of the ticket",
					},
					{
						Name:        "Headline",
						Description: "Markdown description of what happened.",
					},
					{
						Name:        "Key",
						Description: "The issue key. Eg: MM-1234",
					},
					{
						Name:        "ID",
						Description: "Jira issue ID",
					},
				},
				TriggerSetupURL: "/jira" + routeWorkflowTriggerSetup,
			},
		},
		Actions: []workflowclient.ActionParams{
			{
				TypeName:    "create",
				DisplayName: "Jira Create",
				Fields:      []workflowclient.Field{},
				VarInfos:    []workflowclient.VarInfo{},
				URL:         "/jira" + routeWorkflowCreateIssue,
			},
		},
	}

	return respondJSON(w, &params)
}

func httpWorkflowTriggerSetup(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	var params workflowclient.SetupParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "Unable to decode setup params"))
	}

	if params.BaseTrigger.BaseType != "jira_event" {
		return respondErr(w, http.StatusBadRequest,
			errors.New("Unsupported trigger type"))
	}

	var trigger WorkflowTrigger
	if err := json.Unmarshal(params.Trigger, &trigger); err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "Unable to decode trigger"))
	}

	p.workflowTriggerStore.AddTrigger(trigger, params.CallbackURL)

	return http.StatusOK, nil
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

func respondErr(w http.ResponseWriter, code int, err error) (int, error) {
	http.Error(w, err.Error(), code)
	return code, err
}

func respondJSON(w http.ResponseWriter, obj interface{}) (int, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to marshal response"))
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

func (p *Plugin) respondTemplate(w http.ResponseWriter, r *http.Request, contentType string, values interface{}) (int, error) {
	w.Header().Set("Content-Type", contentType)
	t := p.templates[r.URL.Path]
	if t == nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.New("no template found for "+r.URL.Path))
	}
	err := t.Execute(w, values)
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

func (p *Plugin) respondSpecialTemplate(w http.ResponseWriter, key string, status int, contentType string, values interface{}) (int, error) {
	w.Header().Set("Content-Type", contentType)
	t := p.templates[key]
	if t == nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.New("no template found for "+key))
	}
	err := t.Execute(w, values)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}
	return status, nil
}
