// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	routeAutocomplete                           = "/autocomplete"
	routeAutocompleteConnect                    = "/connect"
	routeAutocompleteUserInstance               = "/user-instance"
	routeAutocompleteInstalledInstance          = "/installed-instance"
	routeAutocompleteInstalledInstanceWithAlias = "/installed-instance-with-alias"
	routeAPI                                    = "/api/v2"
	routeAPICreateIssue                         = "/create-issue"
	routeAPIGetCreateIssueMetadata              = "/get-create-issue-metadata-for-project"
	routeAPIGetJiraProjectMetadata              = "/get-jira-project-metadata"
	routeAPIGetSearchIssues                     = "/get-search-issues"
	routeAPIGetAutoCompleteFields               = "/get-search-autocomplete-fields"
	routeAPIGetSearchUsers                      = "/get-search-users"
	routeAPIAttachCommentToIssue                = "/attach-comment-to-issue"
	routeAPIUserInfo                            = "/userinfo"
	routeAPISubscribeWebhook                    = "/webhook"
	routeAPISubscriptionsChannel                = "/subscriptions/channel"
	routeAPISubscriptionsChannelWithID          = routeAPISubscriptionsChannel + "/{id:[A-Za-z0-9]+}"
	routeAPISettingsInfo                        = "/settingsinfo"
	routeIssueTransition                        = "/transition"
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
	routeSharePublicly                          = "/share-issue-publicly"
)

const routePrefixInstance = "instance"

const (
	websocketEventInstanceStatus = "instance_status"
	websocketEventConnect        = "connect"
	websocketEventDisconnect     = "disconnect"
	websocketEventUpdateDefaults = "update_defaults"
)

func (p *Plugin) initializeRouter() {
	p.router = mux.NewRouter()

	p.router.HandleFunc(routeIncomingWebhook, p.handleResponseWithCallbackInstance(p.httpWebhook)).Methods(http.MethodPost)

	// Command autocomplete
	autocompleteRouter := p.router.PathPrefix(routeAutocomplete).Subrouter()
	autocompleteRouter.HandleFunc(routeAutocompleteConnect, p.handleResponse(p.httpAutocompleteConnect)).Methods(http.MethodPost)
	autocompleteRouter.HandleFunc(routeAutocompleteUserInstance, p.handleResponse(p.httpAutocompleteUserInstance)).Methods(http.MethodPost)
	autocompleteRouter.HandleFunc(routeAutocompleteInstalledInstance, p.handleResponse(p.httpAutocompleteInstalledInstance)).Methods(http.MethodPost)
	autocompleteRouter.HandleFunc(routeAutocompleteInstalledInstanceWithAlias, p.handleResponse(p.httpAutocompleteInstalledInstanceWithAlias)).Methods(http.MethodPost)

	apiRouter := p.router.PathPrefix(routeAPI).Subrouter()

	apiRouter.HandleFunc(routeAPIGetAutoCompleteFields, p.handleResponse(p.httpGetAutoCompleteFields)).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPICreateIssue, p.handleResponse(p.httpCreateIssue)).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeAPIGetCreateIssueMetadata, p.handleResponse(p.httpGetCreateIssueMetadataForProjects)).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIGetJiraProjectMetadata, p.handleResponse(p.httpGetJiraProjectMetadata)).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIGetSearchIssues, p.handleResponse(p.httpGetSearchIssues)).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIGetSearchUsers, p.handleResponse(p.httpGetSearchUsers)).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIAttachCommentToIssue, p.handleResponse(p.httpAttachCommentToIssue)).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeIssueTransition, p.handleResponse(p.httpTransitionIssuePostAction)).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeSharePublicly, p.handleResponse(p.httpShareIssuePublicly)).Methods(http.MethodPost)

	// User APIs
	apiRouter.HandleFunc(routeAPIUserInfo, p.handleResponse(p.httpGetUserInfo)).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPISettingsInfo, p.handleResponse(p.httpGetSettingsInfo)).Methods(http.MethodGet)

	// Atlassian Connect application
	p.router.HandleFunc(routeACJSON, p.handleResponseWithCallbackInstance(p.httpACJSON)).Methods(http.MethodPost)
	p.router.HandleFunc(routeACInstalled, p.handleResponse(p.httpACInstalled)).Methods(http.MethodPost)
	p.router.HandleFunc(routeACUninstalled, p.handleResponse(p.httpACUninstalled)).Methods(http.MethodPost)

	// Atlassian Connect user mapping
	p.router.HandleFunc(routeACUserRedirectWithToken, p.handleResponseWithCallbackInstance(p.httpACUserRedirect)).Methods(http.MethodPost)
	p.router.HandleFunc(routeACUserConfirm, p.handleResponseWithCallbackInstance(p.httpACUserInteractive)).Methods(http.MethodPost)
	p.router.HandleFunc(routeACUserConnected, p.handleResponseWithCallbackInstance(p.httpACUserInteractive)).Methods(http.MethodPost)
	p.router.HandleFunc(routeACUserDisconnected, p.handleResponseWithCallbackInstance(p.httpACUserInteractive)).Methods(http.MethodPost)

	// Oauth1 (Jira Server)
	p.router.HandleFunc(routeOAuth1Complete, p.handleResponseWithCallbackInstance(p.httpOAuth1aComplete)).Methods(http.MethodPost)
	p.router.HandleFunc(routeUserDisconnect, p.handleResponseWithCallbackInstance(p.httpOAuth1aDisconnect)).Methods(http.MethodPost)

	// User connect/disconnect links
	p.router.HandleFunc(routeUserConnect, p.handleResponseWithCallbackInstance(p.httpUserConnect)).Methods(http.MethodPost)
	p.router.HandleFunc(routeUserStart, p.handleResponseWithCallbackInstance(p.httpUserStart)).Methods(http.MethodGet)
	p.router.HandleFunc(routeAPIUserDisconnect, p.handleResponse(p.httpUserDisconnect)).Methods(http.MethodPost)

	// Firehose webhook setup for channel subscriptions
	apiRouter.HandleFunc(routeAPISubscribeWebhook, p.handleResponseWithCallbackInstance(p.httpSubscribeWebhook)).Methods(http.MethodPost)

	// Channel Subscriptions
	apiRouter.HandleFunc(routeAPISubscriptionsChannelWithID, p.handleResponse(p.httpChannelSubscriptions)).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPISubscriptionsChannel, p.handleResponse(p.httpChannelSubscriptions)).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeAPISubscriptionsChannel, p.handleResponse(p.httpChannelSubscriptions)).Methods(http.MethodPut)
	apiRouter.HandleFunc(routeAPISubscriptionsChannelWithID, p.handleResponse(p.httpChannelSubscriptions)).Methods(http.MethodDelete)
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/plugins/servlet") {
		body, _ := httputil.DumpRequest(r, true)
		p.errorf("Received an unknown request as a Jira plugin:\n%s", string(body))
	}
	p.router.ServeHTTP(w, r)
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

func (p *Plugin) handleResponse(fn func(w http.ResponseWriter, r *http.Request) (int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := fn(w, r)
		switch {
		case err == nil && status == http.StatusOK:
			p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
		case status == 0:
			p.API.LogDebug("Passed to another router: ", "Path", r.URL.Path, "Method", r.Method)
		case err != nil:
			p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
		default:
			p.API.LogDebug("unexpected plugin response", "Status", strconv.Itoa(status), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
		}
	}
}

func (p *Plugin) handleResponseWithCallbackInstance(fn func(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceURL, _ := splitInstancePath(r.URL.Path)

		callbackInstanceID, err := p.ResolveWebhookInstanceURL(instanceURL)
		if err != nil {
			_, _ = respondErr(w, http.StatusInternalServerError, err)
			return
		}

		status, err := fn(w, r, callbackInstanceID)
		switch {
		case err == nil && status == http.StatusOK:
			p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
		case status == 0:
			p.API.LogDebug("Passed to another router: ", "Path", r.URL.Path, "Method", r.Method)
		case err != nil:
			p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
		default:
			p.API.LogDebug("unexpected plugin response", "Status", strconv.Itoa(status), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
		}
	}
}
