// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
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
	routeInstancePath                           = "/instance/{id}"
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
	routeOAuth2Complete                         = "/oauth2/complete.html"
)

const routePrefixInstance = "instance"

const (
	websocketEventInstanceStatus = "instance_status"
	websocketEventConnect        = "connect"
	websocketEventDisconnect     = "disconnect"
	websocketEventUpdateDefaults = "update_defaults"
)

func makeAutocompleteRoute(path string) string {
	return routeAutocomplete + path
}

func makeAPIRoute(path string) string {
	return routeAPI + path
}

func (p *Plugin) initializeRouter() {
	p.router = mux.NewRouter()
	p.router.Use(p.withRecovery)

	instanceRouter := p.router.PathPrefix(routeInstancePath).Subrouter()
	p.router.HandleFunc(routeIncomingWebhook, p.handleResponseWithCallbackInstance(p.httpWebhook)).Methods(http.MethodPost)

	// Command autocomplete
	autocompleteRouter := p.router.PathPrefix(routeAutocomplete).Subrouter()
	autocompleteRouter.HandleFunc(routeAutocompleteConnect, p.checkAuth(p.handleResponse(p.httpAutocompleteConnect))).Methods(http.MethodGet)
	autocompleteRouter.HandleFunc(routeAutocompleteUserInstance, p.checkAuth(p.handleResponse(p.httpAutocompleteUserInstance))).Methods(http.MethodGet)
	autocompleteRouter.HandleFunc(routeAutocompleteInstalledInstance, p.checkAuth(p.handleResponse(p.httpAutocompleteInstalledInstance))).Methods(http.MethodGet)
	autocompleteRouter.HandleFunc(routeAutocompleteInstalledInstanceWithAlias, p.checkAuth(p.handleResponse(p.httpAutocompleteInstalledInstanceWithAlias))).Methods(http.MethodGet)

	apiRouter := p.router.PathPrefix(routeAPI).Subrouter()

	apiRouter.HandleFunc(routeAPIGetAutoCompleteFields, p.checkAuth(p.handleResponse(p.httpGetAutoCompleteFields))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPICreateIssue, p.checkAuth(p.handleResponse(p.httpCreateIssue))).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeAPIGetCreateIssueMetadata, p.checkAuth(p.handleResponse(p.httpGetCreateIssueMetadataForProjects))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIGetJiraProjectMetadata, p.checkAuth(p.handleResponse(p.httpGetJiraProjectMetadata))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIGetSearchIssues, p.checkAuth(p.handleResponse(p.httpGetSearchIssues))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIGetSearchUsers, p.checkAuth(p.handleResponse(p.httpGetSearchUsers))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIAttachCommentToIssue, p.checkAuth(p.handleResponse(p.httpAttachCommentToIssue))).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeIssueTransition, p.handleResponse(p.httpTransitionIssuePostAction)).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeSharePublicly, p.handleResponse(p.httpShareIssuePublicly)).Methods(http.MethodPost)

	// User APIs
	apiRouter.HandleFunc(routeAPIUserInfo, p.checkAuth(p.handleResponse(p.httpGetUserInfo))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPISettingsInfo, p.checkAuth(p.handleResponse(p.httpGetSettingsInfo))).Methods(http.MethodGet)

	// Atlassian Connect application
	instanceRouter.HandleFunc(routeACJSON, p.handleResponseWithCallbackInstance(p.httpACJSON)).Methods(http.MethodGet)
	p.router.HandleFunc(routeACInstalled, p.handleResponse(p.httpACInstalled)).Methods(http.MethodPost)
	p.router.HandleFunc(routeACUninstalled, p.handleResponse(p.httpACUninstalled)).Methods(http.MethodPost)

	// Atlassian Connect user mapping
	instanceRouter.HandleFunc(routeACUserRedirectWithToken, p.handleResponseWithCallbackInstance(p.httpACUserRedirect)).Methods(http.MethodGet)
	instanceRouter.HandleFunc(routeACUserConfirm, p.handleResponseWithCallbackInstance(p.httpACUserInteractive)).Methods(http.MethodGet)
	instanceRouter.HandleFunc(routeACUserConnected, p.handleResponseWithCallbackInstance(p.httpACUserInteractive)).Methods(http.MethodGet)
	instanceRouter.HandleFunc(routeACUserDisconnected, p.handleResponseWithCallbackInstance(p.httpACUserInteractive)).Methods(http.MethodGet)

	// Oauth1 (Jira Server)
	instanceRouter.HandleFunc(routeOAuth1Complete, p.checkAuth(p.handleResponseWithCallbackInstance(p.httpOAuth1aComplete))).Methods(http.MethodGet)
	instanceRouter.HandleFunc(routeUserDisconnect, p.checkAuth(p.handleResponseWithCallbackInstance(p.httpOAuth1aDisconnect))).Methods(http.MethodGet)

	// OAuth2 (Jira Cloud)
	instanceRouter.HandleFunc(routeOAuth2Complete, p.handleResponseWithCallbackInstance(p.httpOAuth2Complete)).Methods(http.MethodGet)

	// User connect/disconnect links
	instanceRouter.HandleFunc(routeUserConnect, p.checkAuth(p.handleResponseWithCallbackInstance(p.httpUserConnect))).Methods(http.MethodGet)
	p.router.HandleFunc(routeUserStart, p.checkAuth(p.handleResponseWithCallbackInstance(p.httpUserStart))).Methods(http.MethodGet)
	p.router.HandleFunc(routeAPIUserDisconnect, p.checkAuth(p.handleResponse(p.httpUserDisconnect))).Methods(http.MethodPost)

	// Firehose webhook setup for channel subscriptions
	instanceRouter.HandleFunc(makeAPIRoute(routeAPISubscribeWebhook), p.handleResponseWithCallbackInstance(p.httpSubscribeWebhook)).Methods(http.MethodPost)

	// Channel Subscriptions
	apiRouter.HandleFunc(routeAPISubscriptionsChannelWithID, p.checkAuth(p.handleResponse(p.httpChannelGetSubscriptions))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPISubscriptionsChannel, p.checkAuth(p.handleResponse(p.httpChannelCreateSubscription))).Methods(http.MethodPost)
	apiRouter.HandleFunc(routeAPISubscriptionsChannel, p.checkAuth(p.handleResponse(p.httpChannelEditSubscription))).Methods(http.MethodPut)
	apiRouter.HandleFunc(routeAPISubscriptionsChannelWithID, p.checkAuth(p.handleResponse(p.httpChannelDeleteSubscription))).Methods(http.MethodDelete)
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
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

func (p *Plugin) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				p.client.Log.Warn("Recovered from a panic",
					"url", r.URL.String(),
					"error", x,
					"stack", string(debug.Stack()))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) checkAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}

func (p *Plugin) handleResponse(fn func(w http.ResponseWriter, r *http.Request) (int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := fn(w, r)

		p.logResponse(r, status, err)
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

		p.logResponse(r, status, err)
	}
}

func (p *Plugin) logResponse(r *http.Request, status int, err error) {
	if status == 0 || status == http.StatusOK {
		return
	}
	if err != nil {
		p.client.Log.Warn("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
	}

	if status != http.StatusOK {
		p.client.Log.Debug("unexpected plugin response", "Status", strconv.Itoa(status), "Path", r.URL.Path, "Method", r.Method, "query", r.URL.Query().Encode())
	}
}
