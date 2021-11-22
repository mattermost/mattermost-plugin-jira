// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	goexpvar "expvar"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	routeAutocompleteConnect                    = "/autocomplete/connect"
	routeAutocompleteUserInstance               = "/autocomplete/user-instance"
	routeAutocompleteInstalledInstance          = "/autocomplete/installed-instance"
	routeAutocompleteInstalledInstanceWithAlias = "/autocomplete/installed-instance-with-alias"
	routeAPICreateIssue                         = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata              = "/api/v2/get-create-issue-metadata-for-project"
	routeAPIGetJiraProjectMetadata              = "/api/v2/get-jira-project-metadata"
	routeAPIGetSearchIssues                     = "/api/v2/get-search-issues"
	routeAPIGetAutoCompleteFields               = "/api/v2/get-search-autocomplete-fields"
	routeAPIGetSearchUsers                      = "/api/v2/get-search-users"
	routeAPIAttachCommentToIssue                = "/api/v2/attach-comment-to-issue"
	routeAPIUserInfo                            = "/api/v2/userinfo"
	routeAPISubscribeWebhook                    = "/api/v2/webhook"
	routeAPISubscriptionsChannel                = "/api/v2/subscriptions/channel"
	routeAPISettingsInfo                        = "/api/v2/settingsinfo"
	routeAPIStats                               = "/api/v2/stats"
	routeIssueTransition                        = "/api/v2/transition"
	routeAPIUserDisconnect                      = "/api/v3/disconnect"
	routeACInstalled                            = "/ac/installed"
	routeACJSON                                 = "/ac/atlassian-connect.json"
	routeACUninstalled                          = "/ac/uninstalled"
	routeACUserRedirectWithToken                = "/ac/user_redirect.html" // #nosec G101
	routeACUserConfirm                          = "/ac/user_confirm.html"
	routeACUserConnected                        = "/ac/user_connected.html"
	routeACUserDisconnected                     = "/ac/user_disconnected.html"
	routeIncomingWebhook                        = "/webhook"
	routeOAuth1Complete                         = "/oauth1/complete.html"
	routeUserStart                              = "/user/start"
	routeUserConnect                            = "/user/connect"
	routeUserDisconnect                         = "/user/disconnect"
	routeSharePublicly                          = "/api/v2/share-issue-publicly"
)

const routePrefixInstance = "instance"

const (
	websocketEventInstanceStatus = "instance_status"
	websocketEventConnect        = "connect"
	websocketEventDisconnect     = "disconnect"
	websocketEventUpdateDefaults = "update_defaults"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/plugins/servlet") {
		body, _ := httputil.DumpRequest(r, true)
		p.errorf("Received an unknown request as a Jira plugin:\n%s", string(body))
	}

	status, err := p.serveHTTP(c, w, r)
	if err != nil {
		p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
	}
	p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
}

func (p *Plugin) serveHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var err error
	path := r.URL.Path
	instanceURL := ""
	instanceURL, path = splitInstancePath(path)
	callbackInstanceID := types.ID("")

	// Add any "callback" URLs here so that the caller instance is properly identified
	switch path {
	case routeACJSON,
		routeACUserRedirectWithToken,
		routeACUserConfirm,
		routeACUserConnected,
		routeACUserDisconnected,
		routeIncomingWebhook,
		routeOAuth1Complete,
		routeUserDisconnect,
		routeUserConnect,
		routeUserStart,
		routeAPISubscribeWebhook:

		callbackInstanceID, err = p.ResolveWebhookInstanceURL(instanceURL)
		if err != nil {
			return respondErr(w, http.StatusInternalServerError, err)
		}
	}

	switch path {
	// Issue APIs
	case routeAPICreateIssue:
		return p.httpCreateIssue(w, r)
	case routeAPIGetCreateIssueMetadata:
		return p.httpGetCreateIssueMetadataForProjects(w, r)
	case routeAPIGetJiraProjectMetadata:
		return p.httpGetJiraProjectMetadata(w, r)
	case routeAPIGetSearchIssues:
		return p.httpGetSearchIssues(w, r)
	case routeAPIGetAutoCompleteFields:
		return p.httpGetAutoCompleteFields(w, r)
	case routeAPIGetSearchUsers:
		return p.httpGetSearchUsers(w, r)
	case routeAPIAttachCommentToIssue:
		return p.httpAttachCommentToIssue(w, r)
	case routeIssueTransition:
		return p.httpTransitionIssuePostAction(w, r)
	case routeSharePublicly:
		return p.httpShareIssuePublicly(w, r)

	// User APIs
	case routeAPIUserInfo:
		return p.httpGetUserInfo(w, r)
	case routeAPISettingsInfo:
		return p.httpGetSettingsInfo(w, r)

	// Stats
	case routeAPIStats:
		return p.httpAPIStats(w, r)

		// Atlassian Connect application
	case routeACJSON:
		return p.httpACJSON(w, r, callbackInstanceID)
	case routeACInstalled:
		return p.httpACInstalled(w, r)
	case routeACUninstalled:
		return p.httpACUninstalled(w, r)

	// Atlassian Connect user mapping
	case routeACUserRedirectWithToken:
		return p.httpACUserRedirect(w, r, callbackInstanceID)
	case routeACUserConfirm,
		routeACUserConnected,
		routeACUserDisconnected:
		return p.httpACUserInteractive(w, r, callbackInstanceID)

		// Command autocomplete
	case routeAutocompleteConnect:
		return p.httpAutocompleteConnect(w, r)
	case routeAutocompleteUserInstance:
		return p.httpAutocompleteUserInstance(w, r)
	case routeAutocompleteInstalledInstance:
		return p.httpAutocompleteInstalledInstance(w, r)
	case routeAutocompleteInstalledInstanceWithAlias:
		return p.httpAutocompleteInstalledInstanceWithAlias(w, r)

	// Incoming webhook
	case routeIncomingWebhook:
		return p.httpWebhook(w, r, callbackInstanceID)

	// Oauth1 (Jira Server)
	case routeOAuth1Complete:
		return p.httpOAuth1aComplete(w, r, callbackInstanceID)
	case routeUserDisconnect:
		return p.httpOAuth1aDisconnect(w, r, callbackInstanceID)

	// User connect/disconnect links
	case routeUserConnect:
		return p.httpUserConnect(w, r, callbackInstanceID)
	case routeUserStart:
		return p.httpUserStart(w, r, callbackInstanceID)
	case routeAPIUserDisconnect:
		return p.httpUserDisconnect(w, r)

	// Firehose webhook setup for channel subscriptions
	case routeAPISubscribeWebhook:
		return p.httpSubscribeWebhook(w, r, callbackInstanceID)

	// expvar
	case "/debug/vars":
		goexpvar.Handler().ServeHTTP(w, r)
		return 0, nil
	}

	if strings.HasPrefix(path, routeAPISubscriptionsChannel) {
		return p.httpChannelSubscriptions(w, r)
	}

	return respondErr(w, http.StatusNotFound, errors.New("not found"))
}

func (p *Plugin) loadTemplates(dir string) (map[string]*template.Template, error) {
	dir = filepath.Clean(dir)
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
	_, path := splitInstancePath(r.URL.Path)
	w.Header().Set("Content-Type", contentType)
	t := p.templates[path]
	if t == nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.New("no template found for "+path))
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

func instancePath(route string, instanceID types.ID) string {
	encoded := url.PathEscape(encode([]byte(instanceID)))
	return path.Join("/"+routePrefixInstance+"/"+encoded, route)
}

func splitInstancePath(route string) (instanceURL string, remainingPath string) {
	leadingSlash := ""
	ss := strings.Split(route, "/")
	if len(ss) > 1 && ss[0] == "" {
		leadingSlash = "/"
		ss = ss[1:]
	}

	if len(ss) < 2 {
		return "", route
	}
	if ss[0] != routePrefixInstance {
		return "", route
	}

	id, err := decode(ss[1])
	if err != nil {
		return "", route
	}
	return string(id), leadingSlash + strings.Join(ss[2:], "/")
}
