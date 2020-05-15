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

type UserInfo struct {
	IsConnected            bool       `json:"is_connected"`
	CanConnect             bool       `json:"can_connect"`
	User                   *User      `json:"user"`
	Instances              *Instances `json:"instances"`
	DefaultConnectInstance Instance   `json:"default_connect_instance,omitempty"`
	DefaultUseInstance     Instance   `json:"default_use_instance,omitempty"`
}

type User struct {
	MattermostUserID   types.ID   `json:"mattermost_user_id"`
	ConnectedInstances *Instances `json:"connected_instances,omitempty"`
}

type Connection struct {
	jira.User
	PluginVersion      string
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
	Settings           *ConnectionSettings
}

func (c *Connection) JiraAccountID() types.ID {
	if c.AccountID != "" {
		return types.ID(c.AccountID)
	} else {
		return types.ID(c.Name)
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
	connection, err := p.userStore.LoadConnection(instance.GetID(), types.ID(mattermostUserId))
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

	instanceID, err := p.ResolveInstanceID(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// If user is already connected we show them the docs
	connection, err := p.userStore.LoadConnection(instanceID, types.ID(mattermostUserID))
	if err == nil && len(connection.JiraAccountID()) != 0 {
		http.Redirect(w, r, PluginRepo, http.StatusSeeOther)
		return http.StatusSeeOther, nil
	}

	// Otherwise, attempt to connect them
	return p.httpUserConnect(w, r, instanceID)
}

func (p *Plugin) httpGetUserInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	info, err := p.GetUserInfo(types.ID(mattermostUserId))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	return respondJSON(w, info)
}

func (p *Plugin) GetUserInfo(mattermostUserID types.ID) (*UserInfo, error) {
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return nil, err
	}

	user, err := p.MigrateV2User(mattermostUserID)
	if err != nil {
		return nil, err
	}

	isConnected := !user.ConnectedInstances.IsEmpty()
	canConnect := false
	for _, instanceID := range instances.IDs() {
		if !user.ConnectedInstances.Contains(instanceID) {
			canConnect = true
			break
		}
	}

	globalDefaultInstance, _ := p.LoadDefaultInstance("")

	return &UserInfo{
		CanConnect:             canConnect,
		IsConnected:            isConnected,
		Instances:              instances,
		User:                   user,
		DefaultConnectInstance: globalDefaultInstance,
		DefaultUseInstance:     globalDefaultInstance,
	}, nil
}

func (info UserInfo) AsConfigMap() map[string]interface{} {
	m := map[string]interface{}{
		"can_connect":  info.CanConnect,
		"is_connected": info.IsConnected,
	}
	if !info.Instances.IsEmpty() {
		m["instances"] = info.Instances.AsConfigMap()
	}
	if info.User != nil {
		m["user"] = info.User.AsConfigMap()
	}
	if info.DefaultConnectInstance != nil {
		m["default_connect_instance"] = info.DefaultConnectInstance.Common().AsConfigMap()
	}
	if info.DefaultUseInstance != nil {
		m["default_use_instance"] = info.DefaultUseInstance.Common().AsConfigMap()
	}
	return m
}

func (user *User) AsConfigMap() map[string]interface{} {
	return map[string]interface{}{
		"connected_instances": user.ConnectedInstances.AsConfigMap(),
	}
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

func (p *Plugin) connectUser(instance Instance, mattermostUserID types.ID, connection *Connection) error {
	user, err := p.userStore.LoadUser(mattermostUserID)
	if err != nil {
		if err != ErrUserNotFound {
			return err
		}
		user = &User{
			MattermostUserID: mattermostUserID,
		}
	}

	user.ConnectedInstances.Set(instance.Common())

	err = p.userStore.StoreConnection(instance.GetID(), mattermostUserID, connection)
	if err != nil {
		return err
	}
	err = p.userStore.StoreUser(user)
	if err != nil {
		return err
	}

	info, err := p.GetUserInfo(types.ID(mattermostUserID))
	if err != nil {
		return err
	}

	p.API.PublishWebSocketEvent(websocketEventConnect, info.AsConfigMap(),
		&model.WebsocketBroadcast{UserId: mattermostUserID.String()},
	)

	return nil
}

func (p *Plugin) DisconnectUser(instanceID, mattermostUserID types.ID) (*Connection, error) {
	instance, err := p.LoadDefaultInstance(instanceID)
	if err != nil {
		return nil, err
	}
	return p.disconnectUser(instance, mattermostUserID)
}

func (p *Plugin) disconnectUser(instance Instance, mattermostUserID types.ID) (*Connection, error) {
	user, err := p.userStore.LoadUser(mattermostUserID)
	if err != nil {
		return nil, err
	}
	if !user.ConnectedInstances.Contains(instance.GetID()) {
		return nil, ErrInstanceNotFound
	}

	conn, err := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
	if err != nil {
		return nil, err
	}

	user.ConnectedInstances.Delete(instance.GetID())

	err = p.userStore.DeleteConnection(instance.GetID(), mattermostUserID)
	if err != nil {
		return nil, err
	}
	err = p.userStore.StoreUser(user)
	if err != nil {
		return nil, err
	}

	info, err := p.GetUserInfo(types.ID(mattermostUserID))
	if err != nil {
		return nil, err
	}

	p.API.PublishWebSocketEvent(websocketEventDisconnect, info.AsConfigMap(),
		&model.WebsocketBroadcast{UserId: mattermostUserID.String()})
	return conn, nil
}
