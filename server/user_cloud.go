// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"path"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	argJiraJWT       = "jwt"
	argMMToken       = "mm_token"
	cookieSecretName = "jira_temp_cookie"
)

func (p *Plugin) httpACUserRedirect(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	ci, ok := instance.(*cloudInstance)
	if !ok {
		return respondErr(w, http.StatusInternalServerError,
			errors.Errorf("Not supported for instance type %s", instance.Common().Type))
	}

	_, _, err = ci.parseHTTPRequestJWT(r)
	if err != nil {
		return respondErr(w, http.StatusBadRequest, err)
	}

	submitURL := path.Join(ci.Plugin.GetPluginURLPath(), instancePath(routeACUserConfirm, instanceID))

	return ci.Plugin.respondTemplate(w, r, "text/html", struct {
		SubmitURL  string
		ArgJiraJWT string
		ArgMMToken string
	}{
		SubmitURL:  submitURL,
		ArgJiraJWT: argJiraJWT,
		ArgMMToken: argMMToken,
	})
}

func (p *Plugin) httpACUserInteractive(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	ci, ok := instance.(*cloudInstance)
	if !ok {
		return respondErr(w, http.StatusInternalServerError,
			errors.Errorf("2 Not supported for instance type %s", instance.Common().Type))
	}

	jwtToken, _, err := ci.parseHTTPRequestJWT(r)
	if err != nil {
		return respondErr(w, http.StatusBadRequest, err)
	}
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return respondErr(w, http.StatusBadRequest, errors.New("invalid JWT claims"))
	}
	accountID, ok := claims["sub"].(string)
	if !ok {
		return respondErr(w, http.StatusBadRequest, errors.New("invalid JWT claim sub"))
	}

	jiraClient, _, err := ci.getClientForConnection(&Connection{User: jira.User{AccountID: accountID}})
	if err != nil {
		return respondErr(w, http.StatusBadRequest, errors.Errorf("could not get client for user, err: %v", err))
	}

	jUser, _, err := jiraClient.User.GetSelf()
	if err != nil {
		return respondErr(w, http.StatusBadRequest, errors.Errorf("could not get user info for client, err: %v", err))
	}

	mmToken := r.FormValue(argMMToken)
	connection := &Connection{
		PluginVersion: Manifest.Version,
		User: jira.User{
			AccountID:   accountID,
			Key:         jUser.Key,
			Name:        jUser.Name,
			DisplayName: jUser.DisplayName,
		},
		// Set default settings the first time a user connects
		Settings: &ConnectionSettings{
			Notifications: true,
		},
	}

	secretCookie, err := r.Cookie(cookieSecretName)
	if err != nil {
		siteURL := p.GetSiteURL()
		return respondErr(w, http.StatusUnauthorized, errors.New(
			`Mattermost failed to recognize your user account. `+
				`Please make sure third-party cookies are enabled in your browser settings. You can disable this setting after conntecting your Jira account. `+
				`Please also make sure you are signed into Mattermost at `+siteURL))
	}

	mattermostUserID, secret, err := p.ParseAuthToken(mmToken)
	if err != nil {
		return respondErr(w, http.StatusUnauthorized, err)
	}

	mmuser, err := p.client.User.Get(mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to load user "+mattermostUserID))
	}

	_, urlpath := splitInstancePath(r.URL.Path)
	switch urlpath {
	case routeACUserConnected:
		storedSecret := ""
		storedSecret, err = p.otsStore.LoadOneTimeSecret(mattermostUserID)
		if err != nil {
			return respondErr(w, http.StatusUnauthorized, err)
		}

		parsed := strings.Split(storedSecret, "-")
		if len(parsed) < 2 || parsed[0] != secret || parsed[1] != secretCookie.Value {
			return respondErr(w, http.StatusUnauthorized, errors.New("this link has already been used"))
		}
		err = p.connectUser(ci, types.ID(mattermostUserID), connection)
		if err != nil {
			return respondErr(w, http.StatusInternalServerError, err)
		}
		// TODO For https://github.com/mattermost/mattermost-plugin-jira/issues/149, need a channel ID
		// msg := fmt.Sprintf("You have successfully connected your Jira account (**%s**).", connection.DisplayName)
		// _ = p.client.Post.SendEphemeralPost(mattermostUserID, makePost(p.getUserID(), channelID, msg))

	case routeACUserDisconnected:
		_, err = p.DisconnectUser(ci.InstanceID.String(), types.ID(mattermostUserID))

	case routeACUserConfirm:

	default:
		return respondErr(w, http.StatusInternalServerError,
			errors.New("route "+r.URL.Path+" should be unreachable"))
	}
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	mmDisplayName := mmuser.GetDisplayName(model.ShowFullName)
	userName := mmuser.GetDisplayName(model.ShowUsername)
	if mmDisplayName == userName {
		mmDisplayName = "@" + mmDisplayName
	} else {
		mmDisplayName += " (@" + userName + ")"
	}

	// This set of props should work for all relevant routes/templates
	connectSubmitURL := path.Join(p.GetPluginURLPath(), instancePath(routeACUserConnected, instanceID))
	disconnectSubmitURL := path.Join(p.GetPluginURLPath(), instancePath(routeACUserDisconnected, instanceID))
	return ci.Plugin.respondTemplate(w, r, "text/html", struct {
		ConnectSubmitURL      string
		DisconnectSubmitURL   string
		ArgJiraJWT            string
		ArgMMToken            string
		MMToken               string
		JiraDisplayName       string
		MattermostDisplayName string
	}{
		DisconnectSubmitURL:   disconnectSubmitURL,
		ConnectSubmitURL:      connectSubmitURL,
		ArgJiraJWT:            argJiraJWT,
		ArgMMToken:            argMMToken,
		MMToken:               mmToken,
		JiraDisplayName:       jUser.DisplayName,
		MattermostDisplayName: mmDisplayName,
	})
}
