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

func httpACUserRedirect(jci *jiraCloudInstance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	_, _, err := jci.parseHTTPRequestJWT(r)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return respondWithTemplate(w, r, jci.Plugin.templates, "text/html", struct {
		SubmitURL  string
		ArgJiraJWT string
		ArgMMToken string
	}{
		SubmitURL:  path.Join(jci.Plugin.GetPluginURLPath(), routeACUserConnected),
		ArgJiraJWT: argJiraJWT,
		ArgMMToken: argMMToken,
	})
}

func httpACUserConnect(jci *jiraCloudInstance, w http.ResponseWriter, r *http.Request) (int, error) {
	return jci.userConnect(w, r)
}

func httpACUserDisconnect(jci *jiraCloudInstance, w http.ResponseWriter, r *http.Request) (int, error) {
	return jci.userConnect(w, r)
}

func (jci *jiraCloudInstance) userConnect(w http.ResponseWriter, r *http.Request) (int, error) {
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
	mattermostUserId, err := jci.Plugin.ParseAuthToken(mmToken)
	if err != nil {
		return http.StatusBadRequest, err
	}
	mmuser, appErr := jci.Plugin.API.GetUser(mattermostUserId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load user "+mattermostUserId)
	}

	switch r.URL.Path {
	case routeACUserConnected:
		err = jci.Plugin.StoreUserInfoNotify(jci, mattermostUserId, uinfo)
	case routeACUserDisconnected:
		err = jci.Plugin.DeleteUserInfoNotify(jci, mattermostUserId)
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// This set of props should work for both routes/templates
	return respondWithTemplate(w, r, jci.Plugin.templates, "text/html", struct {
		DisconnectSubmitURL   string
		ArgJiraJWT            string
		ArgMMToken            string
		MMToken               string
		JiraDisplayName       string
		MattermostDisplayName string
	}{
		DisconnectSubmitURL:   path.Join(jci.Plugin.GetPluginURLPath(), routeACUserDisconnected),
		ArgJiraJWT:            argJiraJWT,
		ArgMMToken:            argMMToken,
		MMToken:               mmToken,
		JiraDisplayName:       displayName + " (" + username + ")",
		MattermostDisplayName: mmuser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
	})
}
