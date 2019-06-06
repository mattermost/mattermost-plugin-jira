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

// TODO eliminate
var httpACUserRedirect = []ActionFunc{
	RequireHTTPGet,
	handleACUserRedirect,
}

func handleACUserRedirect(a *Action) error {
	submitURL := path.Join(a.PluginConfig.PluginURLPath, routeACUserConfirm)

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

var httpACUserConfirm = []ActionFunc{
	RequireHTTPGet,
	RequireHTTPCloudJWT,
	RequireMattermostUserId,
	handleACUserInteractive,
}

var httpACUserConnected = []ActionFunc{
	// TODO this is wrong, should be a post
	RequireHTTPGet,
	RequireHTTPCloudJWT,
	RequireMattermostUserId,
	RequireInstance,
	handleACUserInteractive,
}

var httpACUserDisconnected = []ActionFunc{
	// TODO this is wrong, should be a post
	RequireHTTPGet,
	RequireHTTPCloudJWT,
	RequireMattermostUserId,
	RequireMattermostUser,
	RequireInstance,
	RequireJiraUser,
	handleACUserInteractive,
}

func handleACUserInteractive(a *Action) error {
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
	jiraUser := JIRAUser{
		User: jira.User{
			Key:  userKey,
			Name: username,
		},
	}

	encryptSecret, err := a.SecretsStore.EnsureAuthTokenEncryptSecret()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	requestedUserId, secret, err := ParseAuthToken(mmToken, encryptSecret)
	if err != nil {
		return a.RespondError(http.StatusUnauthorized, err)
	}

	if a.MattermostUserId != requestedUserId {
		return a.RespondError(http.StatusUnauthorized, nil, "not authorized, user id does not match link")
	}

	route := a.HTTPRequest.URL.Path
	switch route {
	case routeACUserConnected:
		storedSecret := ""
		storedSecret, err = a.SecretsStore.LoadOneTimeSecret(a.MattermostUserId)
		if err != nil {
			return a.RespondError(http.StatusUnauthorized, err)
		}
		if len(storedSecret) == 0 || storedSecret != secret {
			return a.RespondError(http.StatusUnauthorized, nil, "this link has already been used")
		}
		err = StoreUserInfoNotify(a.API, a.UserStore, a.Instance, a.MattermostUserId, jiraUser)

	case routeACUserDisconnected:
		err = DeleteUserInfoNotify(a.API, a.UserStore, a.Instance, a.MattermostUserId)
		a.Debugf("Deleted and notified: %s %+v", a.MattermostUserId, a.JiraUser)

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
		DisconnectSubmitURL:   path.Join(a.PluginConfig.PluginURLPath, routeACUserDisconnected),
		ConnectSubmitURL:      path.Join(a.PluginConfig.PluginURLPath, routeACUserConnected),
		ArgJiraJWT:            argJiraJWT,
		ArgMMToken:            argMMToken,
		MMToken:               mmToken,
		JiraDisplayName:       displayName + " (" + username + ")",
		MattermostDisplayName: a.MattermostUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
	})
}
