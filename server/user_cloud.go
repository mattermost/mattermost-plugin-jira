// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"path"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

const (
	argJiraJWT = "jwt"
	argMMToken = "mm_token"
)

const requireUserApproval = true

func httpACUserRedirect(jci *jiraCloudInstance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	_, _, err := jci.parseHTTPRequestJWT(r)
	if err != nil {
		return http.StatusBadRequest, err
	}

	submitURL := path.Join(jci.Plugin.GetPluginURLPath(), routeACUserConnected)
	if requireUserApproval {
		submitURL = path.Join(jci.Plugin.GetPluginURLPath(), routeACUserConfirm)
	}

	return jci.Plugin.respondWithTemplate(w, r, "text/html", struct {
		SubmitURL  string
		ArgJiraJWT string
		ArgMMToken string
	}{
		SubmitURL:  submitURL,
		ArgJiraJWT: argJiraJWT,
		ArgMMToken: argMMToken,
	})
}

func httpACUserInteractive(jci *jiraCloudInstance, w http.ResponseWriter, r *http.Request) (int, error) {
	jwtToken, _, err := jci.parseHTTPRequestJWT(r)
	if err != nil {
		return http.StatusBadRequest, err
	}
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return http.StatusBadRequest, errors.New("invalid JWT claims")
	}
	context, ok := claims["context"].(map[string]interface{})
	if !ok {
		return http.StatusBadRequest, errors.New("invalid JWT claim context")
	}
	user, ok := context["user"].(map[string]interface{})
	if !ok {
		return http.StatusBadRequest, errors.New("invalid JWT: no user data")
	}
	userKey, _ := user["userKey"].(string)
	username, _ := user["username"].(string)
	displayName, _ := user["displayName"].(string)

	mmToken := r.Form.Get(argMMToken)
	uinfo := JIRAUser{
		User: jira.User{
			Key:  userKey,
			Name: username,
		},
	}
	mattermostUserId, secret, err := jci.Plugin.ParseAuthToken(mmToken)
	if err != nil {
		return http.StatusUnauthorized, err
	}
	mmuser, appErr := jci.Plugin.API.GetUser(mattermostUserId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load user "+mattermostUserId)
	}

	switch r.URL.Path {
	case routeACUserConnected:
		value := ""
		value, err = jci.Plugin.LoadOneTimeSecret(secret)
		if err != nil {
			return http.StatusUnauthorized, err
		}
		err = jci.Plugin.DeleteOneTimeSecret(secret)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		if len(value) == 0 {
			return http.StatusUnauthorized, errors.New("link expired")
		}

		err = jci.Plugin.StoreUserInfoNotify(jci, mattermostUserId, uinfo)

	case routeACUserDisconnected:
		err = jci.Plugin.DeleteUserInfoNotify(jci, mattermostUserId)

	case routeACUserConfirm:

	default:
		return http.StatusInternalServerError,
			errors.New("route " + r.URL.Path + " should be unreachable")
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// This set of props should work for all relevant routes/templates
	return jci.Plugin.respondWithTemplate(w, r, "text/html", struct {
		ConnectSubmitURL      string
		DisconnectSubmitURL   string
		ArgJiraJWT            string
		ArgMMToken            string
		MMToken               string
		JiraDisplayName       string
		MattermostDisplayName string
	}{
		DisconnectSubmitURL:   path.Join(jci.Plugin.GetPluginURLPath(), routeACUserDisconnected),
		ConnectSubmitURL:      path.Join(jci.Plugin.GetPluginURLPath(), routeACUserConnected),
		ArgJiraJWT:            argJiraJWT,
		ArgMMToken:            argMMToken,
		MMToken:               mmToken,
		JiraDisplayName:       displayName + " (" + username + ")",
		MattermostDisplayName: mmuser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
	})
}
