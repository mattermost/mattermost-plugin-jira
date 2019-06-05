// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	WS_EVENT_CONNECT    = "connect"
	WS_EVENT_DISCONNECT = "disconnect"
)

type JIRAUser struct {
	jira.User
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
	// TODO why is this a pointer?
	Settings *UserSettings
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

type UserInfo struct {
	JIRAUser
	IsConnected       bool   `json:"is_connected"`
	InstanceInstalled bool   `json:"instance_installed"`
	JIRAURL           string `json:"jira_url,omitempty"`
}

func httpUserConnect(a *Action) error {
	// Users shouldn't be able to make multiple connections.
	jiraUser, err := a.UserStore.LoadJIRAUser(a.Instance, a.MattermostUserId)
	if err == nil && len(jiraUser.Key) != 0 {
		return a.RespondError(http.StatusForbidden, nil,
			"Already connected to a JIRA account. Please use /jira disconnect to disconnect.")
	}

	redirectURL, err := a.Instance.GetUserConnectURL(a.PluginConfig, a.SecretsStore, a.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	http.Redirect(a.HTTPResponseWriter, a.HTTPRequest, redirectURL, http.StatusFound)
	a.HTTPStatusCode = http.StatusFound
	return nil
}

func httpUserDisconnect(a *Action) error {
	err := DeleteUserInfoNotify(a.API, a.UserStore, a.Instance, a.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	// TODO replace with template
	return a.RespondPrintf(`
<!DOCTYPE html>
<html>
       <head>
               <script>
                       // window.close();
               </script>
       </head>
       <body>
               <p>Disconnected from Jira. Please close this page.</p>
       </body>
</html>
`)
}

func httpAPIGetUserInfo(a *Action) error {
	resp := UserInfo{}
	if ji, err := a.CurrentInstanceStore.LoadCurrentJIRAInstance(); err == nil {
		resp.InstanceInstalled = true
		resp.JIRAURL = ji.GetURL()
		if jiraUser, err := a.UserStore.LoadJIRAUser(ji, a.MattermostUserId); err == nil {
			resp.JIRAUser = jiraUser
			resp.IsConnected = true
		}
	}

	return a.RespondJSON(resp)
}

func StoreUserInfoNotify(api plugin.API, userStore UserStore, instance Instance,
	mattermostUserId string, jiraUser JIRAUser) error {

	err := userStore.StoreUserInfo(instance, mattermostUserId, jiraUser)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"is_connected":  true,
			"jira_username": jiraUser.Name,
			"jira_url":      instance.GetURL(),
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func DeleteUserInfoNotify(api plugin.API, userStore UserStore, instance Instance, mattermostUserId string) error {
	err := userStore.DeleteUserInfo(instance, mattermostUserId)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func UserSettingsNotifications(userStore UserStore, instance Instance, mattermostUserId string,
	jiraUser *JIRAUser, value bool) (string, error) {

	if jiraUser.Settings == nil {
		jiraUser.Settings = &UserSettings{}
	}
	jiraUser.Settings.Notifications = value
	err := userStore.StoreUserInfo(instance, mattermostUserId, *jiraUser)
	if err != nil {
		return "", errors.WithMessage(err, "Could not store new settings. Please contact your system administrator")
	}

	return fmt.Sprintf("Settings updated. Notifications %t.", value), nil
}
