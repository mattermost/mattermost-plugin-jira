// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
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

func (p *Plugin) handleHTTPGetUserInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp := UserInfo{}
	jiraUser, err := p.LoadJIRAUser(ji, mattermostUserId)
	if err == nil {
		resp = UserInfo{
			JIRAUser:    jiraUser,
			IsConnected: true,
			JIRAURL:     ji.GetURL(),
		}
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
	fmt.Println(string(b))
	return http.StatusOK, nil
}

func (p *Plugin) StoreAndNotifyUserInfo(ji Instance, mattermostUserId string, jiraUser JIRAUser) error {
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
