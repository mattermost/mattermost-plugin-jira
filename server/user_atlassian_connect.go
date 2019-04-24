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

const argMMToken = "mm_token"

func httpACUserConfig(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return http.StatusBadRequest, errors.New("Must be a JIRA Cloud instance, is " + ji.GetType())
	}

	_, tokenString, err := jci.parseHTTPRequestJWT(r)
	if err != nil {
		return http.StatusBadRequest, err
	}

	w.Header().Set("Content-Type", "text/html")
	err = p.userConfigTemplate.ExecuteTemplate(w, "config",
		struct {
			SubmitURL  string
			JWT        string
			ArgMMToken string
		}{
			SubmitURL:  path.Join(p.GetPluginURLPath(), routeACUserConfigSubmit),
			JWT:        tokenString,
			ArgMMToken: argMMToken,
		})
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}

	return http.StatusOK, nil
}

func httpACUserConfigSubmit(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return http.StatusBadRequest, errors.New("Must be a JIRA Cloud instance, is " + ji.GetType())
	}

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

	mattermostUserId, err := p.ParseAuthToken(mmToken)
	if err != nil {
		return http.StatusBadRequest, err
	}
	err = p.StoreAndNotifyUserInfo(ji, mattermostUserId, uinfo)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	mmuser, appErr := p.API.GetUser(mattermostUserId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load user "+mattermostUserId)
	}
	w.Header().Set("Content-Type", "text/html")
	_, err = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
    <head>
        <link rel="stylesheet" href="https://unpkg.com/@atlaskit/css-reset@2.0.0/dist/bundle.css" media="all">
	<script src="https://connect-cdn.atl-paas.net/all.js" data-options=""></script>
    </head>
    <body>
    granted Mattermost user ` + mmuser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME) + " (" + mmuser.Username + `) access to JIRA as ` + displayName + " (" + username + `)
    </body>
</html>`))
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}

	return http.StatusOK, nil
}
