// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"path"

	"github.com/dghubble/oauth1"

	"github.com/mattermost/mattermost-server/model"
)

func httpOAuth1Complete(a *Action) error {
	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(a.HTTPRequest)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to parse callback request from Jira")
	}

	requestSecret, err := a.Plugin.LoadOneTimeSecret(requestToken)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	err = a.Plugin.DeleteOneTimeSecret(requestToken)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	oauth1Config, err := a.JiraServerInstance.GetOAuth1Config(a)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to obtain oauth1 config")
	}

	accessToken, accessSecret, err := oauth1Config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to obtain oauth1 access token")
	}

	jiraUser := JIRAUser{
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
	}

	jiraClient, err := a.JiraServerInstance.GetJIRAClient(a, &jiraUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	juser, _, err := jiraClient.User.GetSelf()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	jiraUser.User = *juser
	err = a.Plugin.StoreUserInfoNotify(a.Instance, a.MattermostUserId, jiraUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	a.Plugin.debugf("Stored and notified: %s %+v", a.MattermostUserId, jiraUser)

	return a.RespondTemplate(a.HTTPRequest.URL.Path, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       juser.DisplayName + " (" + juser.Name + ")",
		MattermostDisplayName: a.MattermostUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
		RevokeURL:             path.Join(a.Plugin.GetPluginURLPath(), routeUserDisconnect),
	})
}
