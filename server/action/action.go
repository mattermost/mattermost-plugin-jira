// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/mattermost/mattermost-plugin-jira/store"
)

type Runner interface {
	Run(a Action, ac *ActionContext) error
}

type Action interface {
	FormValue(string) string
	ActionResponder
	Logger
}

type ActionResponder interface {
	RespondTemplate(templateKey, contentType string, values interface{}) error
	RespondJSON(value interface{}) error
	RespondRedirect(redirectURL string) error
	RespondError(httpStatusCode int, err error, wrap ...interface{}) error
	RespondPrintf(format string, args ...interface{}) error
}

type ActionContext struct {
	ActionRouter         *ActionRouter
	API                  plugin.API
	CurrentInstanceStore server.CurrentInstanceStore
	Instance             Instance
	InstanceStore        InstanceStore
	JiraUser             *JiraUser
	JiraClient           *jira.Client
	LogErr               error
	MattermostUser       *model.User
	MattermostUserId     string
	PluginConfig         Config
	PluginContext        *plugin.Context
	SecretStore          SecretStore
	UpstreamJWT          *jwt.Token
	UpstreamRawJWT       string
	UserStore            UserStore
}

// c is a read/write pointer. If it is needed in a goroutine, please clone or
// make separate pointers for the needed values.
type ActionFunc func(a Action, ac *ActionContext) error

type actionHandler struct {
	run      ActionFunc
	metadata interface{}
}

type action struct {
	*ActionContext
}

func (router *ActionRouter) NewAction(p *Plugin, c *plugin.Context, mattermostUserId string) (*action, *ActionContext) {
	ac := &ActionContext{
		API:                  p.API,
		CurrentInstanceStore: p.CurrentInstanceStore,
		InstanceStore:        p.InstanceStore,
		SecretStore:          p.SecretStore,
		UserStore:            p.UserStore,
		PluginConfig:         p.Config,
		PluginContext:        c,
		MattermostUserId:     mattermostUserId,
	}

	return &action{
		ActionContext: ac,
	}, ac
}

func RequireMattermostUserId(a Action, ac *ActionContext) error {
	if ac.MattermostUserId == "" {
		return a.RespondError(http.StatusUnauthorized, nil,
			"not authorized")
	}
	// MattermostUserId is set by the protocol-specific New...Action, nothing to do here
	return nil
}

func RequireMattermostUser(a Action, ac *ActionContext) error {
	if ac.MattermostUser != nil {
		return nil
	}
	err := ActionScript{RequireMattermostUserId}.Run(a, ac)
	if err != nil {
		return err
	}

	mattermostUser, appErr := ac.API.GetUser(ac.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load Mattermost user Id:%s", ac.MattermostUserId)
	}
	ac.MattermostUser = mattermostUser
	return nil
}

func RequireMattermostSysAdmin(a Action, ac *ActionContext) error {
	err := ActionScript{RequireMattermostUser}.Run(a, ac)
	if err != nil {
		return err
	}

	if !ac.MattermostUser.IsInRole(model.SYSTEM_ADMIN_ROLE_ID) {
		return a.RespondError(http.StatusUnauthorized, nil,
			"reserverd for system administrators")
	}
	return nil
}

func RequireJiraUser(a Action, ac *ActionContext) error {
	if ac.JiraUser != nil {
		return nil
	}
	err := ActionScript{RequireMattermostUserId, RequireInstance}.Run(a, ac)
	if err != nil {
		return err
	}

	jiraUser, err := ac.UserStore.LoadJiraUser(ac.Instance, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusUnauthorized, err)
	}
	a.Debugf("action: loaded Jira user %q", jiraUser.Name)
	ac.JiraUser = &jiraUser
	return nil
}

func RequireJiraClient(a Action, ac *ActionContext) error {
	if ac.JiraClient != nil {
		return nil
	}
	err := ActionScript{RequireInstance, RequireJiraUser}.Run(a, ac)
	if err != nil {
		return err
	}

	jiraClient, err := ac.Instance.GetClient(ac.PluginConfig, ac.SecretStore, ac.JiraUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.JiraClient = jiraClient
	a.Debugf("action: loaded Jira client for %q", ac.JiraUser.Name)
	return nil
}

func RequireInstance(a Action, ac *ActionContext) error {
	if ac.Instance != nil {
		return nil
	}
	instance, err := ac.CurrentInstanceStore.LoadCurrentInstance()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.Instance = instance
	a.Debugf("action: loaded Jira instance %q", instance.GetURL())
	return nil
}

type Logger interface {
	Debugf(f string, args ...interface{})
	Infof(f string, args ...interface{})
	Errorf(f string, args ...interface{})
}

func (a action) Debugf(f string, args ...interface{}) {
	a.API.LogDebug(fmt.Sprintf(f, args...))
}

func (a action) Infof(f string, args ...interface{}) {
	a.API.LogInfo(fmt.Sprintf(f, args...))
}

func (a action) Errorf(f string, args ...interface{}) {
	a.API.LogError(fmt.Sprintf(f, args...))
}

func (ar ActionRouter) Run(key string, a Action, ac *ActionContext) {
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
	err := script.Run(a, ac)
	if err != nil {
		return
	}

	// Log
	if ar.LogFilter != nil {
		_ = ar.LogFilter(a, ac)
	}
}

func (script ActionScript) Run(a Action, ac *ActionContext) error {
	for _, f := range script {
		if f == nil {
			continue
		}
		err := f(a, ac)
		if err != nil {
			ac.LogErr = err
			return err
		}
	}
	return nil
}

// func RequireCloudInstance(a Action, ac *ActionContext) error {
// 	if a.JiraCloudInstance != nil {
// 		return nil
// 	}
// 	err := RequireInstance(a, ac)
// 	if err != nil {
// 		return err
// 	}

// 	jci, ok := a.Instance.(*jiraCloudInstance)
// 	if !ok {
// 		return a.RespondError(http.StatusBadRequest, nil, "Must be a Jira Cloud instance, is %s", a.Instance.GetType())
// 	}
// 	a.JiraCloudInstance = jci
// 	a.Debugf("action: loaded Jira cloud instance %v", jci.GetURL())
// 	return nil
// }

// func RequireServerInstance(a Action, ac *ActionContext) error {
// 	if a.JiraServerInstance != nil {
// 		return nil
// 	}
// 	err := RequireInstance(a, ac)
// 	if err != nil {
// 		return err
// 	}

// 	serverInstance, ok := a.Instance.(*jiraServerInstance)
// 	if !ok {
// 		return a.RespondError(http.StatusInternalServerError, nil,
// 			"must be a Jira Server instance, is %s", a.Instance.GetType())
// 	}
// 	a.JiraServerInstance = serverInstance
// 	a.Debugf("action: loaded Jira server instance %v", serverInstance.GetURL())
// 	return nil
// }
