// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	WS_EVENT_CONNECT    = "connect"
	WS_EVENT_DISCONNECT = "disconnect"
)

type User struct {
	MattermostUserID string
	Connections      []*Connection
}

type Connection struct {
	jira.User
	PluginVersion      string
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
	Settings           *ConnectionSettings
}

func (c *Connection) JiraAccountID() string {
	if c.AccountID != "" {
		return c.AccountID
	} else {
		return c.Name
	}
}

type ConnectionSettings struct {
	Notifications bool `json:"notifications"`
}

func (s ConnectionSettings) String() string {
	notifications := "off"
	if s.Notifications {
		notifications = "on"
	}
	return fmt.Sprintf("\tNotifications: %s", notifications)
}

type Info struct {
	User      *User
	Instances []*Instance
}

func (p *Plugin) httpUserConnect(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	instance, err := p.LoadDefaultInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// Users shouldn't be able to make multiple connections.
	connection, err := p.userStore.LoadConnection(instance, mattermostUserId)
	if err == nil && len(connection.JiraAccountID()) != 0 {
		return respondErr(w, http.StatusBadRequest,
			errors.New("You already have a Jira account linked to your Mattermost account. Please use `/jira disconnect` to disconnect."))
	}

	redirectURL, err := instance.GetUserConnectURL(mattermostUserId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
	return http.StatusFound, nil
}

// TODO succinctly document the difference between start and connect
func (p *Plugin) httpUserStart(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	instance, err := p.LoadDefaultInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// If user is already connected we show them the docs
	connection, err := p.userStore.LoadConnection(instance, mattermostUserID)
	if err == nil && len(connection.JiraAccountID()) != 0 {
		http.Redirect(w, r, PluginRepo, http.StatusSeeOther)
		return http.StatusSeeOther, nil
	}

	// Otherwise, attempt to connect them
	return p.httpUserConnect(w, r, instanceID)
}

func (p *Plugin) httpGetInfo(w http.ResponseWriter, r *http.Request) (int, error) {
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

func getUserInfo(p *Plugin, mattermostUserId string) *Info {
	resp := &Info{}

	// if instance, err := p.currentInstanceStore.LoadCurrentJIRAInstance(); err == nil {
	// 	resp.InstanceInstalled = true
	// 	resp.InstanceType = instance.Common().Type
	// 	resp.InstanceDetails = instance.GetDisplayDetails()
	// 	resp.JIRAURL = instance.GetURL()
	// 	if jiraUser, err := instance.GetPlugin().userStore.LoadJIRAUser(instance, mattermostUserId); err == nil {
	// 		resp.Connection = jiraUser
	// 		resp.IsConnected = true
	// 	}
	// }
	return resp
}

func (p *Plugin) httpGetSettingsInfo(w http.ResponseWriter, r *http.Request) (int, error) {
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

func (p *Plugin) connectUser(instance Instance, mattermostUserId string, connection *Connection) error {
	err := p.userStore.StoreConnection(instance, mattermostUserId, connection)
	if err != nil {
		return err
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"is_connected": true,
			"jira_url":     instance.GetURL(),
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func (p *Plugin) DisconnectUser(instanceID types.ID, mattermostUserID string) (*Connection, error) {
	instance, err := p.LoadDefaultInstance(instanceID)
	if err != nil {
		return nil, err
	}
	return p.disconnectUser(instance, mattermostUserID)
}

func (p *Plugin) disconnectUser(instance Instance, mattermostUserId string) (*Connection, error) {
	conn, err := p.userStore.LoadConnection(instance, mattermostUserId)
	if err != nil {
		return nil, err
	}

	err = p.userStore.DeleteConnection(instance, mattermostUserId)
	if err != nil {
		return nil, err
	}

	instance.Common().API.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		map[string]interface{}{
			"is_connected": false,
			"jira_url":     instance.GetURL(),
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return conn, nil
}
