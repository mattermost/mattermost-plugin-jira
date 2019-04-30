// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"

	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func httpOAuth1Complete(jsi *jiraServerInstance, w http.ResponseWriter, r *http.Request) (int, error) {
	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to parse callback request from JIRA")
	}

	requestSecret, err := jsi.Plugin.LoadOneTimeSecret(requestToken)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = jsi.Plugin.DeleteOneTimeSecret(requestToken)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	mattermostUserId := r.Header.Get("Mattermost-User-ID")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	oauth1Config, err := jsi.GetOAuth1Config()
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to obtain oauth1 config")
	}

	accessToken, accessSecret, err := oauth1Config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to obtain oauth1 access token")
	}

	jiraUser := JIRAUser{
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
	}

	jiraClient, err := jsi.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	user, _, err := jiraClient.User.GetSelf()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	jiraUser.User = *user

	err = jsi.Plugin.StoreUserInfoNotify(jsi, mattermostUserId, jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	html := `
<!DOCTYPE html>
<html>
	<head>
		<script>
			window.close();
		</script>
	</head>
	<body>
		<p>Completed connecting to JIRA. Please close this page.</p>
	</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	_, err = w.Write([]byte(html))
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

func httpOAuth1PublicKey(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
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

	rsaKey, err := p.EnsureRSAKey()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	b, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to encode public key")
	}

	pemkey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}

	w.Header().Set("Content-Type", "text/plain")
	_, err = w.Write(pem.EncodeToMemory(pemkey))
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}
