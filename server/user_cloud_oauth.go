// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func (p *Plugin) httpOAuth2Configure(w http.ResponseWriter, r *http.Request) (status int, err error) {
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be POST"))
	}

	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	authorized, err := authorizedSysAdmin(p, mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	if !authorized {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to decode request"))
	}

	var config CloudOAuthConfigure
	err = json.Unmarshal(body, &config)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to unmarshal request"))
	}

	_, _, err = p.installCloudOAuthInstance(config.InstanceURL, config.ClientID, config.ClientSecret)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "unable to configure cloud oauth"))
	}

	return respondJSON(w, []string{"OK"})
}

func (p *Plugin) httpOAuth2Complete(w http.ResponseWriter, r *http.Request, instanceID types.ID) (status int, err error) {
	code := r.URL.Query().Get("code")
	if code == "" {
		return respondErr(w, http.StatusBadRequest, errors.New("Bad request: missing code"))
	}
	state := r.URL.Query().Get("state")
	if state == "" {
		return respondErr(w, http.StatusBadRequest, errors.New("Bad request: missing state"))
	}

	if len(strings.Split(state, "_")) != 2 || strings.Split(state, "_")[1] == "" {
		return respondErr(w, http.StatusBadRequest, errors.New("Bad request: invalid state"))
	}

	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	oAuthInstance, ok := instance.(*cloudOAuthInstance)
	if !ok {
		return respondErr(w, http.StatusInternalServerError, errors.Errorf("Not supported for instance type %s", instance.Common().Type))
	}

	oAuthConf := oAuthInstance.GetOAuthConfig()

	stateSecret := strings.Split(state, "_")[0]
	mattermostUserID := strings.Split(state, "_")[1]
	storedSecret, err := p.otsStore.LoadOneTimeSecret(mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusUnauthorized, errors.New("state not found or might be expired"))
	}
	parsed := strings.Split(storedSecret, "_")
	if len(parsed) < 2 || parsed[0] != stateSecret {
		return respondErr(w, http.StatusUnauthorized, errors.New("state token mismatch"))
	}

	mmUser, appErr := p.API.GetUser(mattermostUserID)
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(appErr, "failed to load user "+mattermostUserID))
	}

	token, err := oAuthConf.Exchange(context.Background(), code)
	if err != nil {
		p.client.Log.Error("error while exchanging authorization code for access token", "error", err)
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(err, "error while exchanging authorization code for access token"))
	}

	connection := &Connection{
		PluginVersion: manifest.Version,
		OAuth2Token:   token,
	}

	client, err := instance.GetClient(connection)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	juser, err := client.GetSelf()
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	connection.User = *juser

	// Set default settings the first time a user connects
	connection.Settings = &ConnectionSettings{Notifications: true}

	err = p.connectUser(instance, types.ID(mattermostUserID), connection)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	return p.respondTemplate(w, r, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       juser.DisplayName + " (" + juser.Name + ")",
		MattermostDisplayName: mmUser.GetDisplayName(model.ShowNicknameFullName),
		RevokeURL:             path.Join(p.GetPluginURLPath(), instancePath(routeUserDisconnect, instance.GetID())),
	})
}
