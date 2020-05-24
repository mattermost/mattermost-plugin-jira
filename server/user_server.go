// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"path"
	"strings"

	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type OAuth1aTemporaryCredentials struct {
	Token  string
	Secret string
}

func (p *Plugin) httpOAuth1aComplete(w http.ResponseWriter, r *http.Request, instanceID types.ID) (status int, err error) {
	// Prettify error output
	defer func() {
		if err == nil {
			return
		}

		errtext := err.Error()
		if len(errtext) > 0 {
			errtext = strings.ToUpper(errtext[:1]) + errtext[1:]
		}
		status, err = p.respondSpecialTemplate(w, "/other/message.html", status, "text/html", struct {
			Header  string
			Message string
		}{
			Header:  "Failed to connect to Jira.",
			Message: errtext,
		})
	}()

	instance, err := p.LoadDefaultInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	si, ok := instance.(*serverInstance)
	if !ok {
		return respondErr(w, http.StatusInternalServerError,
			errors.Errorf("Not supported for instance type %s", instance.Common().Type))
	}

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to parse callback request from Jira"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-ID")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}
	mmuser, appErr := p.API.GetUser(mattermostUserId)
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load user "+mattermostUserId))
	}

	oauthTmpCredentials, err := p.otsStore.OneTimeLoadOauth1aTemporaryCredentials(mattermostUserId)
	if err != nil || oauthTmpCredentials == nil || len(oauthTmpCredentials.Token) <= 0 {
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to get temporary credentials for "+mattermostUserId))
	}

	if oauthTmpCredentials.Token != requestToken {
		return respondErr(w, http.StatusUnauthorized, errors.New("request token mismatch"))
	}

	// Although we pass the oauthTmpCredentials as required here. The JIRA server does not appar to validate it.
	// We perform the check above for reuse so this is irrelavent to the security from our end.
	accessToken, accessSecret, err := si.getOAuth1Config().AccessToken(requestToken, oauthTmpCredentials.Secret, verifier)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to obtain oauth1 access token"))
	}

	connection := &Connection{
		PluginVersion:      manifest.Version,
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
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

	err = p.connectUser(instance, types.ID(mattermostUserId), connection)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	return p.respondTemplate(w, r, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       juser.DisplayName + " (" + juser.Name + ")",
		MattermostDisplayName: mmuser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
		RevokeURL:             path.Join(p.GetPluginURLPath(), instancePath(routeUserDisconnect, instance.GetID())),
	})
}

func (p *Plugin) httpOAuth1aDisconnect(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	instance, err := p.LoadDefaultInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	_, err = p.disconnectUser(instance, types.ID(mattermostUserId))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	return p.respondSpecialTemplate(w, "/other/message.html", http.StatusOK,
		"text/html", struct {
			Header  string
			Message string
		}{
			Header:  "Disconnected from Jira.",
			Message: "It is now safe to close this browser window.",
		})
}

func httpOAuth1aPublicKey(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}

	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	if !p.API.HasPermissionTo(userID, model.PERMISSION_MANAGE_SYSTEM) {
		return respondErr(w, http.StatusForbidden, errors.New("forbidden"))
	}

	w.Header().Set("Content-Type", "text/plain")
	pkey, err := publicKeyString(p)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to load public key"))
	}
	_, err = w.Write(pkey)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response"))
	}
	return http.StatusOK, nil
}

func publicKeyString(p *Plugin) ([]byte, error) {
	rsaKey := p.getConfig().rsaKey
	b, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to encode public key")
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}
