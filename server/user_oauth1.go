// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/dghubble/oauth1"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) handleHTTPOAuth1Connect(w http.ResponseWriter, r *http.Request) (int, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jis, ok := ji.(*jiraServerInstance)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("Must be a JIRA Server instance")
	}

	oauth1Config, err := jis.GetOAuth1Config()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	requestToken, requestSecret, err := oauth1Config.RequestToken()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = p.StoreOAuth1RequestToken(requestToken, requestSecret)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	authURL, err := oauth1Config.AuthorizationURL(requestToken)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, authURL.String(), http.StatusFound)
	return http.StatusFound, nil
}

func (p *Plugin) handleHTTPOAuth1Complete(w http.ResponseWriter, r *http.Request) (int, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jis, ok := ji.(*jiraServerInstance)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("Must be a JIRA Server instance")
	}

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	requestSecret, err := p.LoadOAuth1RequestToken(requestToken)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = p.DeleteOAuth1RequestToken(requestToken)

	mattermostUserId := r.Header.Get("Mattermost-User-ID")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	oauth1Config, err := jis.GetOAuth1Config()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	accessToken, accessSecret, err := oauth1Config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser := JIRAUser{
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	}

	user, _, err := jiraClient.User.GetSelf()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get current user: %v", err)
	}
	jiraUser.User = *user

	err = p.StoreAndNotifyUserInfo(ji, mattermostUserId, jiraUser)
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
	w.Write([]byte(html))
	return http.StatusOK, nil
}

func (p *Plugin) handleHTTPOAuth1PublicKey(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	if !p.API.HasPermissionTo(userID, model.PERMISSION_MANAGE_SYSTEM) {
		return http.StatusForbidden, fmt.Errorf("Forbidden")
	}

	rsaKey, err := p.EnsureRSAKey()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	b, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	pemkey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(pem.EncodeToMemory(pemkey))
	return http.StatusOK, nil
}
