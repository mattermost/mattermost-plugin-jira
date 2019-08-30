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

	"github.com/mattermost/mattermost-server/model"
)

type OAuth1aTemporaryCredentials struct {
	Token  string
	Secret string
}

func httpOAuth1aComplete(jsi *jiraServerInstance, w http.ResponseWriter, r *http.Request) (status int, err error) {
	// Prettify error output
	defer func() {
		if err == nil {
			return
		}

		errtext := err.Error()
		if len(errtext) > 0 {
			errtext = strings.ToUpper(errtext[:1]) + errtext[1:]
		}
		status, err = jsi.Plugin.respondSpecialTemplate(w, "/other/message.html", status, "text/html", struct {
			Header  string
			Message string
		}{
			Header:  "Failed to connect to Jira.",
			Message: errtext,
		})
	}()

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to parse callback request from Jira")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-ID")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}
	mmuser, appErr := jsi.Plugin.API.GetUser(mattermostUserId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load user "+mattermostUserId)
	}

	oauthTmpCredentials, err := jsi.Plugin.otsStore.OneTimeLoadOauth1aTemporaryCredentials(mattermostUserId)
	if err != nil || oauthTmpCredentials == nil || len(oauthTmpCredentials.Token) <= 0 {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to get temporary credentials for "+mattermostUserId)
	}

	if oauthTmpCredentials.Token != requestToken {
		return http.StatusUnauthorized, errors.New("request token mismatch")
	}

	oauth1Config, err := jsi.GetOAuth1Config()
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to obtain oauth1 config")
	}

	// Although we pass the oauthTmpCredentials as required here. The JIRA server does not appar to validate it.
	// We perform the check above for reuse so this is irrelavent to the security from our end.
	accessToken, accessSecret, err := oauth1Config.AccessToken(requestToken, oauthTmpCredentials.Secret, verifier)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to obtain oauth1 access token")
	}

	jiraUser := JIRAUser{
		PluginVersion:      manifest.Version,
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
	}

	client, err := jsi.GetClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	juser, err := client.GetSelf()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	jiraUser.User = *juser

	// Set default settings the first time a user connects
	jiraUser.Settings = &UserSettings{Notifications: true}

	err = jsi.Plugin.StoreUserInfoNotify(jsi, mattermostUserId, jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return jsi.Plugin.respondWithTemplate(w, r, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       juser.DisplayName + " (" + juser.Name + ")",
		MattermostDisplayName: mmuser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
		RevokeURL:             path.Join(jsi.Plugin.GetPluginURLPath(), routeUserDisconnect),
	})
}

func httpOAuth1aDisconnect(ji *jiraServerInstance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	err := ji.GetPlugin().userDisconnect(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return ji.GetPlugin().respondSpecialTemplate(w, "/other/message.html", http.StatusOK,
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
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	if !p.API.HasPermissionTo(userID, model.PERMISSION_MANAGE_SYSTEM) {
		return http.StatusForbidden, errors.New("forbidden")
	}

	w.Header().Set("Content-Type", "text/plain")
	pkey, err := publicKeyString(p)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to load public key")
	}
	_, err = w.Write(pkey)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

func publicKeyString(p *Plugin) ([]byte, error) {
	rsaKey, err := p.secretsStore.EnsureRSAKey()
	if err != nil {
		return nil, err
	}

	b, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to encode public key")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}
