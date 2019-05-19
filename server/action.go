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

type Action struct {
	// Always there
	Plugin        *Plugin
	PluginConfig  config
	PluginContext *plugin.Context

	// Input
	CommandHeader *model.CommandArgs
	CommandArgs   []string
	HTTPRequest   *http.Request

	// Output
	CommandResponse    *model.CommandResponse
	HTTPResponseWriter http.ResponseWriter
	HTTPStatusCode     int

	// Variables
	Instance         Instance
	MattermostUserId string
	MattermostUser   *model.User
	JiraUser         *JIRAUser
	JiraClient       *jira.Client

	// Server-specific
	JiraServerInstance *jiraServerInstance

	// Cloud-specific
	JiraCloudInstance *jiraCloudInstance
	JiraJWT           *jwt.Token
	JiraRawJWT        string
}

type ActionScript struct {
	Filters []ActionFunc
	Handler ActionFunc
}

type ActionRouter struct {
	RouteHandlers       map[string]*ActionScript
	DefaultRouteHandler ActionFunc
	Log                 []ActionFunc
}

func NewAction(p *Plugin, c *plugin.Context) *Action {
	return &Action{
		Plugin:        p,
		PluginContext: c,
		PluginConfig:  p.getConfig(),
	}
}

func RequireMattermostUser(a *Action) error {
	if a.MattermostUser != nil {
		return nil
	}
	if a.MattermostUserId == "" {
		return a.RespondError(http.StatusInternalServerError, nil,
			"misconfiguration: required MattermostUserId missing")
	}

	mmuser, appErr := a.Plugin.API.GetUser(a.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load Mattermost user Id:%s", a.MattermostUserId)
	}

	a.MattermostUser = mmuser
	a.Plugin.debugf("action: loaded Mattermost user %v", mmuser.GetDisplayName(""))
	return nil
}

func RequireMattermostSysAdmin(a *Action) error {
	if !strings.Contains(a.MattermostUser.Roles, "system_admin") {
		return a.RespondError(http.StatusUnauthorized, nil,
			"reserverd for system administrators")
	}
	return nil
}

func RequireJiraUser(a *Action) error {
	if a.JiraUser != nil {
		return nil
	}
	if a.MattermostUserId == "" {
		return a.RespondError(http.StatusInternalServerError, nil,
			"misconfiguration: required MattermostUserId missing")
	}
	err := RequireInstance(a)
	if err != nil {
		return err
	}

	jiraUser, err := a.Plugin.LoadJIRAUser(a.Instance, a.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusUnauthorized, err)
	}
	a.Plugin.debugf("action: loaded Jira user %v", jiraUser.Name)
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

	jiraClient, err := a.Instance.GetJIRAClient(a, nil)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	a.JiraClient = jiraClient
	a.Plugin.debugf("action: loaded Jira client")
	return nil
}

func RequireInstance(a *Action) error {
	if a.Instance != nil {
		return nil
	}
	ji, err := a.Plugin.LoadCurrentJIRAInstance()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	a.Instance = ji
	a.Plugin.debugf("action: loaded Jira instance %v", ji.GetURL())
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
		return a.RespondError(http.StatusBadRequest, nil, "Must be a JIRA Cloud instance, is %s", a.Instance.GetType())
	}
	a.JiraCloudInstance = jci
	a.Plugin.debugf("action: loaded Jira cloud instance %v", jci.GetURL())
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

	jsi, ok := a.Instance.(*jiraServerInstance)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil,
			"must be a Jira Server instance, is %s", a.Instance.GetType())
	}
	a.JiraServerInstance = jsi
	a.Plugin.debugf("action: loaded Jira server instance %v", jsi.GetURL())
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
	a.Plugin.debugf("action: verified request method %v", method)
	return nil
}

func RequireHTTPMattermostUserId(a *Action) error {
	if a.MattermostUserId != "" {
		return nil
	}
	mattermostUserId := a.HTTPRequest.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return a.RespondError(http.StatusUnauthorized, nil,
			"not authorized")
	}
	a.MattermostUserId = mattermostUserId
	a.Plugin.debugf("action: found MattermostUserId %v", mattermostUserId)
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
	a.Plugin.debugf("action: verified Jira JWT")
	return nil
}

func RequireCommandMattermostUserId(a *Action) error {
	if a.MattermostUserId != "" {
		return nil
	}
	mattermostUserId := a.CommandHeader.UserId
	if mattermostUserId == "" {
		return a.RespondError(http.StatusUnauthorized, nil,
			"not authorized")
	}
	a.MattermostUserId = mattermostUserId
	a.Plugin.debugf("action: found MattermostUserId %v", mattermostUserId)
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

func (a *Action) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := a.Plugin.templates[templateKey]
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

func (ar ActionRouter) Run(key string, a *Action) {
	// See if we have a script for the key, otherwise run the default
	script := ar.RouteHandlers[key]
	if script == nil {
		script = &ActionScript{
			Handler: ar.DefaultRouteHandler,
		}
	}

	// A helper to run a []ActionFunc
	runList := func(list []ActionFunc) error {
		for _, f := range list {
			err := f(a)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Run the script
	func() {
		err := runList(script.Filters)
		if err != nil {
			return
		}
		if script.Handler != nil {
			err = script.Handler(a)
			if err != nil {
				return
			}
		}
	}()

	// Log
	runList(ar.Log)
}
