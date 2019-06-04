// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

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
	Settings           *UserSettings
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

func httpUserConnect(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	// Users shouldn't be able to make multiple connections.
	if jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId); err == nil && len(jiraUser.Key) != 0 {
		return http.StatusBadRequest, errors.New("Already connected to a JIRA account. Please use /jira disconnect to disconnect.")
	}

	redirectURL, err := ji.GetUserConnectURL(mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
	return http.StatusFound, nil
}

func httpUserDisconnect(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
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

	html := `
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
`

	w.Header().Set("Content-Type", "text/html")
	_, err = w.Write([]byte(html))
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}

	return http.StatusOK, nil
}

func httpAPIGetUserInfo(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	resp := UserInfo{}
	if ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance(); err == nil {
		if jiraUser, err := p.userStore.LoadJIRAUser(ji, mattermostUserId); err == nil {
			resp = UserInfo{
				JIRAUser:          jiraUser,
				InstanceInstalled: true,
				IsConnected:       true,
				JIRAURL:           ji.GetURL(),
			}
		}
	}

	b, _ := json.Marshal(resp)
	_, err := w.Write(b)
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

func (p *Plugin) StoreUserInfoNotify(ji Instance, mattermostUserId string, jiraUser JIRAUser) error {
	err := p.userStore.StoreUserInfo(ji, mattermostUserId, jiraUser)
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
	err := p.userStore.DeleteUserInfo(ji, mattermostUserId)
	if err != nil {
		return err
	}

	ji.GetPlugin().API.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func (p *Plugin) userDisconnect(ji Instance, mattermostUserId string) error {
	if err := p.DeleteUserInfoNotify(ji, mattermostUserId); err != nil {
		return err
	}
	return nil
}
