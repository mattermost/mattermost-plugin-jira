// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type ActionFunc func(a *Action) error

type ActionScript []ActionFunc

type Action struct {
	// Always there
	// Plugin        *Plugin
	API                  plugin.API
	CurrentInstanceStore CurrentInstanceStore
	InstanceStore        InstanceStore
	SecretsStore         SecretsStore
	UserStore            UserStore
	PluginConfig         Config
	PluginContext        *plugin.Context

	// Input
	CommandHeader *model.CommandArgs
	CommandArgs   []string
	HTTPRequest   *http.Request

	// Output
	CommandResponse    *model.CommandResponse
	HTTPResponseWriter http.ResponseWriter
	HTTPStatusCode     int
	LogErr             error

	// Variables
	Instance         Instance
	MattermostUserId string
	MattermostUser   *model.User
	JiraUser         *JiraUser
	JiraClient       *jira.Client

	// Server-specific
	JiraServerInstance *jiraServerInstance

	// Cloud-specific
	JiraCloudInstance *jiraCloudInstance
	JiraJWT           *jwt.Token
	JiraRawJWT        string
}

type ActionRouter struct {
	RouteHandlers       map[string]ActionScript
	DefaultRouteHandler ActionFunc
	Log                 ActionFunc
}

func NewAction(p *Plugin, c *plugin.Context) *Action {
	return &Action{
		API:                  p.API,
		CurrentInstanceStore: p.CurrentInstanceStore,
		InstanceStore:        p.InstanceStore,
		SecretsStore:         p.SecretsStore,
		UserStore:            p.UserStore,
		PluginConfig:         p.Config,
		PluginContext:        c,
	}
}

func RequireMattermostUserId(a *Action) error {
	if a.MattermostUserId != "" {
		return nil
	}
	mattermostUserId := ""
	if a.CommandHeader != nil && a.CommandHeader.UserId != "" {
		mattermostUserId = a.CommandHeader.UserId
	} else if a.HTTPRequest != nil {
		mattermostUserId = a.HTTPRequest.Header.Get("Mattermost-User-Id")
	}
	if mattermostUserId == "" {
		return a.RespondError(http.StatusUnauthorized, nil,
			"not authorized")
	}
	a.MattermostUserId = mattermostUserId
	a.Debugf("action: found MattermostUserId %v", mattermostUserId)
	return nil
}

func RequireMattermostUser(a *Action) error {
	if a.MattermostUser != nil {
		return nil
	}
	err := RequireMattermostUserId(a)
	if err != nil {
		return err
	}

	mmuser, appErr := a.API.GetUser(a.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load Mattermost user Id:%s", a.MattermostUserId)
	}

	a.MattermostUser = mmuser
	a.Debugf("action: loaded Mattermost user %v", mmuser.GetDisplayName(""))
	return nil
}

func RequireMattermostSysAdmin(a *Action) error {
	err := ActionScript{RequireMattermostUser, RequireInstance}.Run(a)
	if err != nil {
		return err
	}
	if !a.MattermostUser.IsInRole(model.SYSTEM_ADMIN_ROLE_ID) {
		return a.RespondError(http.StatusUnauthorized, nil,
			"reserverd for system administrators")
	}
	return nil
}

func RequireJiraUser(a *Action) error {
	if a.JiraUser != nil {
		return nil
	}
	err := ActionScript{RequireMattermostUserId, RequireInstance}.Run(a)
	if err != nil {
		return err
	}

	jiraUser, err := a.UserStore.LoadJiraUser(a.Instance, a.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusUnauthorized, err)
	}
	a.Debugf("action: loaded Jira user %v", jiraUser.Name)
	a.JiraUser = &jiraUser
	return nil
}

func RequireJiraClient(a *Action) error {
	if a.JiraClient != nil {
		return nil
	}
	err := RequireJiraUser(a)
	if err != nil {
		return err
	}

	jiraClient, err := a.Instance.GetClient(a.PluginConfig, a.SecretsStore, a.JiraUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	a.JiraClient = jiraClient
	a.Debugf("action: loaded Jira client")
	return nil
}

func RequireInstance(a *Action) error {
	if a.Instance != nil {
		return nil
	}
	instance, err := a.CurrentInstanceStore.LoadCurrentInstance()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	a.Instance = instance
	a.Debugf("action: loaded Jira instance %v", instance.GetURL())
	return nil
}

func RequireCloudInstance(a *Action) error {
	if a.JiraCloudInstance != nil {
		return nil
	}
	err := RequireInstance(a)
	if err != nil {
		return err
	}

	jci, ok := a.Instance.(*jiraCloudInstance)
	if !ok {
		return a.RespondError(http.StatusBadRequest, nil, "Must be a Jira Cloud instance, is %s", a.Instance.GetType())
	}
	a.JiraCloudInstance = jci
	a.Debugf("action: loaded Jira cloud instance %v", jci.GetURL())
	return nil
}

func RequireServerInstance(a *Action) error {
	if a.JiraServerInstance != nil {
		return nil
	}
	err := RequireInstance(a)
	if err != nil {
		return err
	}

	serverInstance, ok := a.Instance.(*jiraServerInstance)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil,
			"must be a Jira Server instance, is %s", a.Instance.GetType())
	}
	a.JiraServerInstance = serverInstance
	a.Debugf("action: loaded Jira server instance %v", serverInstance.GetURL())
	return nil
}

func RequireHTTPGet(a *Action) error {
	return requireHTTPMethod(a, http.MethodGet)
}

func RequireHTTPPost(a *Action) error {
	return requireHTTPMethod(a, http.MethodPost)
}

func requireHTTPMethod(a *Action, method string) error {
	if a.HTTPRequest.Method != method {
		return a.RespondError(http.StatusMethodNotAllowed, nil,
			"method %s is not allowed, must be %s", a.HTTPRequest.Method, method)
	}
	a.Debugf("action: verified request method %v", method)
	return nil
}

func RequireHTTPCloudJWT(a *Action) error {
	if a.JiraJWT != nil {
		return nil
	}
	err := RequireCloudInstance(a)
	if err != nil {
		return err
	}

	err = a.HTTPRequest.ParseForm()
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err,
			"failed to parse HTTP equest")
	}

	tokenString := a.HTTPRequest.Form.Get("jwt")
	if tokenString == "" {
		return a.RespondError(http.StatusBadRequest, nil,
			"no jwt found in the HTTP request")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf(
				"unsupported signing method: %v", token.Header["alg"])
		}
		// HMAC secret is a []byte
		return []byte(a.JiraCloudInstance.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return a.RespondError(http.StatusUnauthorized, err,
			"failed to validatte JWT")
	}

	a.JiraJWT = token
	a.JiraRawJWT = tokenString
	a.Debugf("action: verified Jira JWT")
	return nil
}

func (a *Action) RespondError(httpStatusCode int, err error, wrap ...interface{}) error {
	if len(wrap) > 0 {
		fmt := wrap[0].(string)
		if err != nil {
			err = errors.WithMessagef(err, fmt, wrap[1:]...)
		} else {
			err = errors.Errorf(fmt, wrap[1:]...)
		}
	}

	if err == nil {
		return nil
	}

	if a.HTTPResponseWriter != nil {
		a.HTTPStatusCode = httpStatusCode
		http.Error(a.HTTPResponseWriter, err.Error(), httpStatusCode)
	} else {
		a.CommandResponse = commandResponse(err.Error())
	}

	return err
}

func (a *Action) RespondPrintf(format string, args ...interface{}) error {
	text := fmt.Sprintf(format, args...)
	if a.HTTPResponseWriter != nil {
		a.HTTPResponseWriter.Header().Set("Content-Type", "text/plain")

		_, err := a.HTTPResponseWriter.Write([]byte(text))
		if err != nil {
			return a.RespondError(http.StatusInternalServerError, err,
				"failed to write response")
		}
	} else {
		a.CommandResponse = commandResponse(text)
	}
	return nil
}

func (a *Action) RespondRedirect(redirectURL string) error {
	if a.HTTPResponseWriter != nil {
		status := http.StatusFound
		if a.HTTPRequest.Method != http.MethodGet {
			status = http.StatusTemporaryRedirect
		}
		http.Redirect(a.HTTPResponseWriter, a.HTTPRequest, redirectURL, status)
		a.HTTPStatusCode = status
	} else {
		a.CommandResponse = &model.CommandResponse{
			GotoLocation: redirectURL,
		}
	}
	return nil
}

func (a *Action) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := a.PluginConfig.Templates[templateKey]
	if t == nil {
		return a.RespondError(http.StatusInternalServerError, nil,
			"no template found for %q", templateKey)
	}
	if a.HTTPResponseWriter != nil {
		a.HTTPResponseWriter.Header().Set("Content-Type", contentType)
		err := t.Execute(a.HTTPResponseWriter, values)
		if err != nil {
			return a.RespondError(http.StatusInternalServerError, err,
				"failed to write response")
		}
	} else {
		bb := &bytes.Buffer{}
		err := t.Execute(bb, values)
		if err != nil {
			return a.RespondError(http.StatusInternalServerError, err,
				"failed to write response")
		}
		a.CommandResponse = commandResponse(string(bb.Bytes()))
	}
	return nil
}

func (a *Action) RespondJSON(value interface{}) error {
	if a.HTTPResponseWriter != nil {
		a.HTTPResponseWriter.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(a.HTTPResponseWriter).Encode(value)
		if err != nil {
			return a.RespondError(http.StatusInternalServerError, err,
				"failed to write response")
		}
	} else {
		bb, err := json.Marshal(value)
		if err != nil {
			return a.RespondError(http.StatusInternalServerError, err,
				"failed to write response")
		}
		a.CommandResponse = commandResponse(string(bb))
	}
	return nil
}

func (a *Action) Debugf(f string, args ...interface{}) {
	a.API.LogDebug(fmt.Sprintf(f, args...))
}

func (a *Action) Infof(f string, args ...interface{}) {
	a.API.LogInfo(fmt.Sprintf(f, args...))
}

func (a *Action) Errorf(f string, args ...interface{}) {
	a.API.LogError(fmt.Sprintf(f, args...))
}

func (ar ActionRouter) Run(key string, a *Action) {
	key = strings.TrimRight(key, "/")
	// See if we have a script for the exact key match
	script := ar.RouteHandlers[key]
	if script == nil {
		// Look for a subpath match
		script = ar.RouteHandlers[key+"/*"]
	}
	// Look for a /* above
	for script == nil {
		n := strings.LastIndex(key, "/")
		if n == -1 {
			break
		}
		script = ar.RouteHandlers[key[:n]+"/*"]
		key = key[:n]
	}
	// Use the default, if needed
	if script == nil {
		script = ActionScript{
			ar.DefaultRouteHandler,
		}
	}

	// Run the script
	err := script.Run(a)
	if err != nil {
		return
	}

	// Log
	if ar.Log != nil {
		_ = ar.Log(a)
	}
}

func (script ActionScript) Run(a *Action) error {
	for _, f := range script {
		if f == nil {
			continue
		}
		err := f(a)
		if err != nil {
			a.LogErr = err
			return err
		}
	}
	return nil
}
