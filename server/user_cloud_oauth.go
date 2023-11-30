// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const TokenExpiryTimeBufferInMinutes = 5

func (p *Plugin) httpOAuth2Complete(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	code := r.URL.Query().Get("code")
	if code == "" {
		return respondErr(w, http.StatusBadRequest, errors.New("Bad request: missing code"))
	}
	state := r.URL.Query().Get("state")
	if state == "" {
		return respondErr(w, http.StatusBadRequest, errors.New("Bad request: missing state"))
	}

	stateArray := strings.Split(state, "_")
	if len(stateArray) != 2 || stateArray[1] == "" {
		return respondErr(w, http.StatusBadRequest, errors.New("Bad request: invalid state"))
	}

	stateSecret := stateArray[0]
	mattermostUserID := stateArray[1]
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
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(appErr, fmt.Sprintf("failed to load user %s", mattermostUserID)))
	}

	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.Wrap(err, "Error occurred while loading instance"))
	}

	connection, err := p.GenerateInitialOAuthToken(mattermostUserID, code, instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.Wrap(err, "Error occurred while generating initial oauth token"))
	}

	client, err := instance.GetClient(connection)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.Wrap(err, "Error occurred while getting client"))
	}

	jiraUser, err := client.GetSelf()
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.Wrap(err, "Error occurred while getting Jira user"))
	}
	connection.User = *jiraUser

	// Set default settings when the user connects for the first time
	connection.Settings = &ConnectionSettings{Notifications: true}
	connection.MattermostUserID = types.ID(mattermostUserID)

	if err := p.connectUser(instance, types.ID(mattermostUserID), connection); err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.Wrap(err, fmt.Sprintf("Error occurred while connecting user. UserID: %s", mattermostUserID)))
	}

	return p.respondTemplate(w, r, ContentTypeHTML, struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       jiraUser.DisplayName + " (" + jiraUser.Name + ")",
		MattermostDisplayName: mmUser.GetDisplayName(model.ShowNicknameFullName),
		RevokeURL:             path.Join(p.GetPluginURLPath(), instancePath(routeUserDisconnect, instance.GetID())),
	})
}

func (p *Plugin) GenerateInitialOAuthToken(mattermostUserID, code string, instanceID types.ID) (*Connection, error) {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return nil, err
	}
	oAuthInstance, ok := instance.(*cloudOAuthInstance)
	if !ok {
		return nil, errors.Errorf("Not supported for instance type %s", instance.Common().Type)
	}

	oAuthConf := oAuthInstance.GetOAuthConfig()

	token, err := oAuthConf.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", oAuthInstance.CodeVerifier))
	if err != nil {
		p.client.Log.Error("error while exchanging authorization code for access token", "error", err)
		return nil, errors.WithMessage(err, "error while exchanging authorization code for access token")
	}

	connection, err := p.userStore.LoadConnection(instanceID, types.ID(mattermostUserID))
	if err != nil {
		return nil, err
	}

	connection.OAuth2Token = token
	return connection, nil
}
