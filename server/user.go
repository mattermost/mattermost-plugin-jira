// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type User struct {
	PluginVersion      string
	MattermostUserID   types.ID   `json:"mattermost_user_id"`
	ConnectedInstances *Instances `json:"connected_instances,omitempty"`
	DefaultInstanceID  types.ID   `json:"default_instance_id,omitempty"`
}

type Connection struct {
	jira.User
	PluginVersion      string
	Oauth1AccessToken  string        `json:",omitempty"`
	Oauth1AccessSecret string        `json:",omitempty"`
	OAuth2Token        *oauth2.Token `json:",omitempty"`
	Settings           *ConnectionSettings
	DefaultProjectKey  string   `json:"default_project_key,omitempty"`
	MattermostUserID   types.ID `json:"mattermost_user_id"`
}

func (c *Connection) JiraAccountID() types.ID {
	if c.AccountID != "" {
		return types.ID(c.AccountID)
	}

	return types.ID(c.Name)
}

type ConnectionSettings struct {
	Notifications bool `json:"notifications"`
}

func (s *ConnectionSettings) String() string {
	notifications := "off"
	if s != nil && s.Notifications {
		notifications = "on"
	}
	return fmt.Sprintf("\tNotifications: %s", notifications)
}

func NewUser(mattermostUserID types.ID) *User {
	return &User{
		MattermostUserID:   mattermostUserID,
		ConnectedInstances: NewInstances(),
	}
}

func (p *Plugin) httpUserConnect(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// Users shouldn't be able to make multiple connections.
	// TODO <> this block needs to be updated. Though idk if this route will still get called?
	connection, err := p.userStore.LoadConnection(instance.GetID(), types.ID(mattermostUserID))
	if err == nil && len(connection.JiraAccountID()) != 0 {
		return respondErr(w, http.StatusBadRequest,
			errors.New("you already have a Jira account linked to your Mattermost account. Please use `/jira disconnect` to disconnect"))
	}

	redirectURL, cookie, err := instance.GetUserConnectURL(mattermostUserID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	if cookie != nil {
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
	return http.StatusFound, nil
}

func (p *Plugin) httpUserDisconnect(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	disconnectPayload := &struct {
		InstanceID string `json:"instance_id"`
	}{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to decode request"))
	}

	err = json.Unmarshal(body, disconnectPayload)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to unmarshal disconnect payload"))
	}

	_, err = p.DisconnectUser(disconnectPayload.InstanceID, types.ID(mattermostUserID))
	if errors.Cause(err) == kvstore.ErrNotFound {
		return respondErr(w, http.StatusNotFound,
			errors.Errorf(
				"could not complete the **disconnection** request. You do not currently have a Jira account at %q linked to your Mattermost account",
				disconnectPayload.InstanceID))
	}
	if err != nil {
		return respondErr(w, http.StatusNotFound,
			errors.Errorf("could not complete the **disconnection** request. Error: %v", err))
	}

	_, err = w.Write([]byte(`{"success": true}`))
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}

	return http.StatusOK, nil
}

// TODO succinctly document the difference between start and connect
func (p *Plugin) httpUserStart(w http.ResponseWriter, r *http.Request, instanceID types.ID) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")

	// If user is already connected we show them the docs
	connection, err := p.userStore.LoadConnection(instanceID, types.ID(mattermostUserID))
	if err == nil && len(connection.JiraAccountID()) != 0 {
		http.Redirect(w, r, PluginRepo, http.StatusSeeOther)
		return http.StatusSeeOther, nil
	}

	// Otherwise, attempt to connect them
	return p.httpUserConnect(w, r, instanceID)
}

func (user *User) AsConfigMap() map[string]interface{} {
	return map[string]interface{}{
		"mattermost_user_id":  user.MattermostUserID.String(),
		"connected_instances": user.ConnectedInstances.AsConfigMap(),
		"default_instance_id": user.DefaultInstanceID.String(),
	}
}

func (p *Plugin) UpdateUserDefaults(mattermostUserID, instanceID types.ID, projectKey string) {
	user, err := p.userStore.LoadUser(mattermostUserID)
	if err != nil {
		return
	}
	if !user.ConnectedInstances.Contains(instanceID) {
		return
	}

	connection, err := p.userStore.LoadConnection(instanceID, user.MattermostUserID)
	if err != nil {
		return
	}
	if instanceID != "" && instanceID != user.DefaultInstanceID {
		user.DefaultInstanceID = instanceID
		err = p.userStore.StoreUser(user)
		if err != nil {
			return
		}
	}

	if projectKey != "" && projectKey != connection.DefaultProjectKey {
		connection.DefaultProjectKey = projectKey
		err = p.userStore.StoreConnection(instanceID, user.MattermostUserID, connection)
		if err != nil {
			return
		}
	}

	info, err := p.GetUserInfo(mattermostUserID, user)
	if err != nil {
		return
	}

	p.client.Frontend.PublishWebSocketEvent(websocketEventUpdateDefaults, info.AsConfigMap(),
		&model.WebsocketBroadcast{UserId: mattermostUserID.String()},
	)
}

func (p *Plugin) httpGetSettingsInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	conf := p.getConfig()
	return respondJSON(w, struct {
		UIEnabled                              bool `json:"ui_enabled"`
		SecurityLevelEmptyForJiraSubscriptions bool `json:"security_level_empty_for_jira_subscriptions"`
	}{
		UIEnabled:                              conf.EnableJiraUI,
		SecurityLevelEmptyForJiraSubscriptions: conf.SecurityLevelEmptyForJiraSubscriptions,
	})
}

func (p *Plugin) connectUser(instance Instance, mattermostUserID types.ID, connection *Connection) error {
	user, err := p.userStore.LoadUser(mattermostUserID)
	if err != nil {
		if errors.Cause(err) != kvstore.ErrNotFound {
			return err
		}
		user = NewUser(mattermostUserID)
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

	_ = p.setupFlow.ForUser(string(mattermostUserID)).Go(stepConnected)

	info, err := p.GetUserInfo(mattermostUserID, user)
	if err != nil {
		return err
	}

	p.client.Frontend.PublishWebSocketEvent(websocketEventConnect, info.AsConfigMap(),
		&model.WebsocketBroadcast{UserId: mattermostUserID.String()},
	)

	p.TrackUserEvent("userConnected", mattermostUserID.String(), nil)

	return nil
}

func (p *Plugin) DisconnectUser(instanceURL string, mattermostUserID types.ID) (*Connection, error) {
	user, instance, err := p.LoadUserInstance(mattermostUserID, instanceURL)
	if err != nil {
		return nil, err
	}
	return p.disconnectUser(instance, user)
}

func (p *Plugin) disconnectUser(instance Instance, user *User) (*Connection, error) {
	if !user.ConnectedInstances.Contains(instance.GetID()) {
		return nil, errors.Wrapf(kvstore.ErrNotFound, "user is not connected to %q", instance.GetID())
	}
	conn, err := p.userStore.LoadConnection(instance.GetID(), user.MattermostUserID)
	if err != nil {
		return nil, err
	}

	if user.DefaultInstanceID == instance.GetID() {
		user.DefaultInstanceID = ""
	}

	user.ConnectedInstances.Delete(instance.GetID())

	err = p.userStore.DeleteConnection(instance.GetID(), user.MattermostUserID)
	if err != nil && errors.Cause(err) != kvstore.ErrNotFound {
		return nil, err
	}
	err = p.userStore.StoreUser(user)
	if err != nil {
		return nil, err
	}

	info, err := p.GetUserInfo(user.MattermostUserID, user)
	if err != nil {
		return nil, err
	}
	p.client.Frontend.PublishWebSocketEvent(websocketEventDisconnect, info.AsConfigMap(),
		&model.WebsocketBroadcast{UserId: user.MattermostUserID.String()})

	p.TrackUserEvent("userDisconnected", user.MattermostUserID.String(), nil)

	return conn, nil
}
