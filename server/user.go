// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
)

const (
	WS_EVENT_CONNECT    = "connect"
	WS_EVENT_DISCONNECT = "disconnect"
)

type JIRAUserInfo struct {
	// These fields come from JIRA, so their JSON names must not change.
	Key       string `json:"key,omitempty"`
	AccountId string `json:"accountId,omitempty"`
	Name      string `json:"name,omitempty"`
}

type UserInfo struct {
	JIRAUserInfo
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
	jiraUserInfo, err := p.LoadJIRAUserInfo(ji, mattermostUserId)
	if err == nil {
		resp = UserInfo{
			JIRAUserInfo: jiraUserInfo,
			IsConnected:  true,
			JIRAURL:      ji.GetURL(),
		}
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
	fmt.Println(string(b))
	return http.StatusOK, nil
}

func (p *Plugin) StoreAndNotifyUserInfo(ji JIRAInstance, mattermostUserId string, info JIRAUserInfo) error {
	err := p.StoreUserInfo(ji, mattermostUserId, info)
	if err != nil {
		return err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"is_connected":    true,
			"jira_username":   info.Name,
			"jira_account_id": info.AccountId,
			"jira_url":        ji.GetURL(),
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}
