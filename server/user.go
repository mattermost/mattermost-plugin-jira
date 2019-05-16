// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"

	"github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/model"
)

const (
	WS_EVENT_CONNECT    = "connect"
	WS_EVENT_DISCONNECT = "disconnect"
)

type JIRAUser struct {
	jira.User
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
}

type UserInfo struct {
	JIRAUser
	IsConnected bool   `json:"is_connected"`
	JIRAURL     string `json:"jira_url,omitempty"`
}

func httpUserConnect(a *Action) error {
	// Users shouldn't be able to make multiple connections.
	jiraUser, err := a.Plugin.LoadJIRAUser(a.Instance, a.MattermostUserId)
	if err == nil && len(jiraUser.Key) != 0 {
		return a.RespondError(http.StatusForbidden, nil,
			"Already connected to a JIRA account. Please use /jira disconnect to disconnect.")
	}

	redirectURL, err := a.Instance.GetUserConnectURL(a)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	http.Redirect(a.HTTPResponseWriter, a.HTTPRequest, redirectURL, http.StatusFound)
	a.HTTPStatusCode = http.StatusFound
	return nil
}

func httpUserDisconnect(a *Action) error {
	err := a.Plugin.DeleteUserInfoNotify(a.Instance, a.MattermostUserId)
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
	if a.JiraUser != nil {
		resp = UserInfo{
			JIRAUser:    *a.JiraUser,
			IsConnected: true,
			JIRAURL:     a.Instance.GetURL(),
		}
	}

	a.Plugin.debugf("httpAPIGetUserInfo: %+v", resp)
	return a.RespondJSON(resp)
}

func (p *Plugin) StoreUserInfoNotify(ji Instance, mattermostUserId string, jiraUser JIRAUser) error {
	err := p.StoreUserInfo(ji, mattermostUserId, jiraUser)
	if err != nil {
		return err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"is_connected":  true,
			"jira_username": jiraUser.Name,
			"jira_url":      ji.GetURL(),
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func (p *Plugin) DeleteUserInfoNotify(ji Instance, mattermostUserId string) error {
	err := p.DeleteUserInfo(ji, mattermostUserId)
	if err != nil {
		return err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}
