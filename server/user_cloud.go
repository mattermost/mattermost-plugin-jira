// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"path"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-server/model"
)

const (
	argJiraJWT = "jwt"
	argMMToken = "mm_token"
)

func httpACUserRedirect(a *Action) error {
	submitURL := path.Join(a.Plugin.GetPluginURLPath(), routeACUserConfirm)

	return a.RespondTemplate(a.HTTPRequest.URL.Path, "text/html", struct {
		SubmitURL  string
		ArgJiraJWT string
		ArgMMToken string
	}{
		SubmitURL:  submitURL,
		ArgJiraJWT: argJiraJWT,
		ArgMMToken: argMMToken,
	})
}

func httpACUserInteractive(a *Action) error {
	claims, ok := a.JiraJWT.Claims.(jwt.MapClaims)
	if !ok {
		return a.RespondError(http.StatusBadRequest, nil,
			"invalid JWT claims")
	}
	context, ok := claims["context"].(map[string]interface{})
	if !ok {
		return a.RespondError(http.StatusBadRequest, nil,
			"invalid JWT claim context")
	}
	user, ok := context["user"].(map[string]interface{})
	if !ok {
		return a.RespondError(http.StatusBadRequest, nil,
			"invalid JWT: no user data")
	}
	userKey, _ := user["userKey"].(string)
	username, _ := user["username"].(string)
	displayName, _ := user["displayName"].(string)

	mmToken := a.HTTPRequest.Form.Get(argMMToken)
	uinfo := JIRAUser{
		User: jira.User{
			Key:  userKey,
			Name: username,
		},
	}
	mattermostUserId, secret, err := a.Plugin.ParseAuthToken(mmToken)
	if err != nil {
		return a.RespondError(http.StatusUnauthorized, err)
	}
	mmuser, appErr := a.Plugin.API.GetUser(mattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load user %q", mattermostUserId)
	}

	route := a.HTTPRequest.URL.Path
	switch route {
	case routeACUserConnected:
		value := ""
		value, err = a.Plugin.LoadOneTimeSecret(secret)
		if err != nil {
			return a.RespondError(http.StatusUnauthorized, err)
		}
		err = a.Plugin.DeleteOneTimeSecret(secret)
		if err != nil {
			return a.RespondError(http.StatusInternalServerError, err)
		}
		if len(value) == 0 {
			return a.RespondError(http.StatusUnauthorized, nil, "link expired")
		}

		// Set default settings the first time a user connects
		uinfo.Settings = &UserSettings{Notifications: true}

		err = a.Plugin.StoreUserInfoNotify(a.Instance, mattermostUserId, uinfo)
		a.Plugin.debugf("Stored and notified: %s %+v", mattermostUserId, uinfo)

	case routeACUserDisconnected:
		err = a.Plugin.DeleteUserInfoNotify(a.Instance, mattermostUserId)
		a.Plugin.debugf("Deleted and notified: %s %+v", mattermostUserId, uinfo)

	case routeACUserConfirm:

	default:
		return a.RespondError(http.StatusInternalServerError, nil,
			"route %q should be unreachable", route)
	}
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	// This set of props should work for all relevant routes/templates
	return a.RespondTemplate(route, "text/html", struct {
		ConnectSubmitURL      string
		DisconnectSubmitURL   string
		ArgJiraJWT            string
		ArgMMToken            string
		MMToken               string
		JiraDisplayName       string
		MattermostDisplayName string
	}{
		DisconnectSubmitURL:   path.Join(a.Plugin.GetPluginURLPath(), routeACUserDisconnected),
		ConnectSubmitURL:      path.Join(a.Plugin.GetPluginURLPath(), routeACUserConnected),
		ArgJiraJWT:            argJiraJWT,
		ArgMMToken:            argMMToken,
		MMToken:               mmToken,
		JiraDisplayName:       displayName + " (" + username + ")",
		MattermostDisplayName: mmuser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
	})
}
