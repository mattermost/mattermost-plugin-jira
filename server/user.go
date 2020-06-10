// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

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
	// TODO <> this block needs to be updated. Though idk if this route will still get called?
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

func (p *Plugin) httpUserDisconnect(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be POST"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	disconnectPayload := &struct {
		InstanceID string `json:"instance_id"`
	}{}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to decode request"))
	}

	err = json.Unmarshal(body, disconnectPayload)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to unmarshal disconnect payload"))
	}

	instanceID := types.ID(disconnectPayload.InstanceID)
	_, err = p.DisconnectUser(instanceID, types.ID(mattermostUserId))

	if errors.Cause(err) == kvstore.ErrNotFound {
		return respondErr(w, http.StatusNotFound,
			errors.Errorf("Could not complete the **disconnection** request. You do not currently have a Jira account at %q linked to your Mattermost account.", instanceID))
	}
	if err != nil {
		return respondErr(w, http.StatusNotFound,
			errors.Errorf("Could not complete the **disconnection** request. Error: %v", err))
	}

	w.Write([]byte(`{"success": true}`))
	return http.StatusOK, nil
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

func (user *User) AsConfigMap() map[string]interface{} {
	return map[string]interface{}{
		"mattermost_user_id":  user.MattermostUserID.String(),
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
		if errors.Cause(err) != kvstore.ErrNotFound {
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
		return nil, errors.Wrapf(kvstore.ErrNotFound, "user is not connected to %q", instance.GetID())
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
