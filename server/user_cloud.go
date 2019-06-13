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

	submitURL := path.Join(jci.Plugin.GetPluginURLPath(), routeACUserConfirm)

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

	accountId, _ := user["accountId"].(string)
	displayName, _ := user["displayName"].(string)
	userKey, _ := user["userKey"].(string)
	username, _ := user["username"].(string)

	mmToken := r.Form.Get(argMMToken)
	uinfo := JIRAUser{
		UserKey: accountId,
		User: jira.User{
			AccountID:   accountId,
			DisplayName: displayName,
			Key:         userKey,
			Name:        username,
		},
	}
	mattermostUserId := r.Header.Get("Mattermost-User-ID")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New(
			`Mattermost failed to recognize your user account. ` +
				`Please make sure third-party cookies are not disabled in your browser settings.`)
	}

	requestedUserId, secret, err := jci.Plugin.ParseAuthToken(mmToken)
	if err != nil {
		return http.StatusUnauthorized, err
	}

	if mattermostUserId != requestedUserId {
		return http.StatusUnauthorized, errors.New("not authorized, user id does not match link")
	}

	mmuser, appErr := jci.Plugin.API.GetUser(mattermostUserId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load user "+mattermostUserId)
	}

	switch r.URL.Path {
	case routeACUserConnected:
		storedSecret := ""
		storedSecret, err = jci.Plugin.LoadOneTimeSecret(mattermostUserId)
		if err != nil {
			return http.StatusUnauthorized, err
		}
		if len(storedSecret) == 0 || storedSecret != secret {
			return http.StatusUnauthorized, errors.New("this link has already been used")
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

	mmDisplayName := mmuser.GetDisplayName(model.SHOW_FULLNAME)
	userName := mmuser.GetDisplayName(model.SHOW_USERNAME)
	if mmDisplayName == userName {
		mmDisplayName = "@" + mmDisplayName
	} else {
		mmDisplayName += " (@" + userName + ")"
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
		MattermostDisplayName: mmDisplayName,
	})
}
