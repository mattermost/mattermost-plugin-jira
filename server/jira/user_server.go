// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"path"

	"github.com/dghubble/oauth1"

	"github.com/mattermost/mattermost-server/model"
)

type OAuth1aTemporaryCredentials struct {
	Token  string
	Secret string
}

var httpOAuth1Complete = []ActionFunc{
	// TODO Should this be a post? Can it be one (Jira/OAuth1 controls)?
	RequireHTTPGet,
	RequireMattermostUserId,
	RequireMattermostUser,
	RequireInstance,
	RequireServerInstance,
	handleOAuth1Complete,
}

func handleOAuth1Complete(a Action) error {
	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(a.HTTPRequest)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to parse callback request from Jira")
	}

	oauthTmpCredentials, err := a.SecretsStore.OneTimeLoadOauth1aTemporaryCredentials(a.MattermostUserId)
	if err != nil || oauthTmpCredentials == nil || len(oauthTmpCredentials.Token) <= 0 {
		return a.RespondError(http.StatusInternalServerError, err, "failed to get temporary credentials for %q", a.MattermostUserId)
	}

	if oauthTmpCredentials.Token != requestToken {
		return a.RespondError(http.StatusUnauthorized, nil, "request token mismatch")
	}

	oauth1Config, err := a.JiraServerInstance.GetOAuth1Config(a.PluginConfig, a.SecretsStore)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to obtain oauth1 config")
	}

	// Although we pass the oauthTmpCredentials as required here. The Jira server does not appar to validate it.
	// We perform the check above for reuse so this is irrelavent to the security from our end.
	accessToken, accessSecret, err := oauth1Config.AccessToken(requestToken, oauthTmpCredentials.Secret, verifier)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to obtain oauth1 access token")
	}

	jiraUser := JiraUser{
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
	}

	jiraClient, err := a.JiraServerInstance.GetClient(a.PluginConfig, a.SecretsStore, &jiraUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	juser, _, err := jiraClient.User.GetSelf()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	jiraUser.User = *juser
	// Set default settings the first time a user connects
	jiraUser.Settings = &UserSettings{Notifications: true}

	err = StoreUserInfoNotify(a.API, a.UserStore, a.Instance, a.MattermostUserId, jiraUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	a.Debugf("Stored and notified: %s %+v", a.MattermostUserId, jiraUser)

	return a.RespondTemplate(a.HTTPRequest.URL.Path, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       juser.DisplayName + " (" + juser.Name + ")",
		MattermostDisplayName: a.MattermostUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
		RevokeURL:             path.Join(a.PluginConfig.PluginURLPath, routeUserDisconnect),
	})
}

var httpOAuth1PublicKey = []ActionFunc{
	RequireHTTPGet,
	RequireMattermostUserId,
	handleOAuth1PublicKey,
}

func handleOAuth1PublicKey(a Action) error {
	if !a.API.HasPermissionTo(a.MattermostUserId, model.PERMISSION_MANAGE_SYSTEM) {
		return a.RespondError(http.StatusForbidden, nil, "forbidden")
	}

	pkey, err := publicKeyString(a.SecretsStore)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err, "failed to load public key")
	}
	return a.RespondPrintf(string(pkey))
}
