// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	WS_EVENT_CONNECT    = "connect"
	WS_EVENT_DISCONNECT = "disconnect"
)

type JIRAUser struct {
	jira.User
	PluginVersion      string
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
	Settings           *UserSettings
}

func (u JIRAUser) Key() string {
	if u.AccountID != "" {
		return u.AccountID
	} else {
		return u.Name
	}
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

func (us UserSettings) String() string {
	notifications := "off"
	if us.Notifications {
		notifications = "on"
	}
	return fmt.Sprintf("\tNotifications: %s", notifications)
}

type UserInfo struct {
	JIRAUser
	IsConnected       bool              `json:"is_connected"`
	InstanceInstalled bool              `json:"instance_installed"`
	InstanceType      string            `json:"instance_type"`
	JIRAURL           string            `json:"jira_url,omitempty"`
	InstanceDetails   map[string]string `json:"instance_details,omitempty"`
}

func httpUserConnect(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	// Users shouldn't be able to make multiple connections.
	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err == nil && len(jiraUser.Key()) != 0 {
		return respondErr(w, http.StatusBadRequest,
			errors.New("You already have a Jira account linked to your Mattermost account. Please use `/jira disconnect` to disconnect."))
	}

	redirectURL, err := ji.GetUserConnectURL(mattermostUserId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
	return http.StatusFound, nil
}

func httpUserStart(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	// If user is already connected we show them the docs
	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserID)
	if err == nil && len(jiraUser.Key()) != 0 {
		http.Redirect(w, r, PluginRepo, http.StatusSeeOther)
		return http.StatusSeeOther, nil
	}

	// Otherwise, attempt to connect them
	return httpUserConnect(ji, w, r)
}

func httpAPIGetUserInfo(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	return respondJSON(w, getUserInfo(p, mattermostUserId))
}

func getUserInfo(p *Plugin, mattermostUserId string) UserInfo {
	resp := UserInfo{}
	if ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance(); err == nil {
		resp.InstanceInstalled = true
		resp.InstanceType = ji.GetType()
		resp.InstanceDetails = ji.GetDisplayDetails()
		resp.JIRAURL = ji.GetURL()
		if jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId); err == nil {
			resp.JIRAUser = jiraUser
			resp.IsConnected = true
		}
	}
	return resp
}

func httpAPIGetSettingsInfo(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	return respondJSON(w, struct {
		UIEnabled bool `json:"ui_enabled"`
	}{
		UIEnabled: p.getConfig().EnableJiraUI,
	})
}

func (p *Plugin) StoreUserInfoNotify(ji Instance, mattermostUserId string, jiraUser JIRAUser) error {
	err := p.userStore.StoreUserInfo(ji, mattermostUserId, jiraUser)
	if err != nil {
		return err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"is_connected": true,
			"jira_url":     ji.GetURL(),
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
			"jira_url":     ji.GetURL(),
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
