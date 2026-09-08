// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost/server/public/model"

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
	SavedFieldValues   *SavedFieldValues `json:"saved_field_values,omitempty"`
	MattermostUserID   types.ID          `json:"mattermost_user_id"`
}

type SavedFieldValues struct {
	ProjectKey string `json:"project_key,omitempty"`
	IssueType  string `json:"issue_type,omitempty"`
}

func (connection *Connection) JiraAccountID() types.ID {
	if connection.AccountID != "" {
		return types.ID(connection.AccountID)
	}

	return types.ID(connection.Name)
}

type ConnectionSettings struct {
	Notifications          bool `json:"notifications"`
	RolesForDMNotification map[string]bool
}

func (s *ConnectionSettings) String() string {
	assigneeNotifications := "Notifications for assignee: off"
	mentionNotifications := "Notifications for mention: off"
	reporterNotifications := "Notifications for reporter: off"
	watchingNotifications := "Notifications for watching: off"

	if s != nil && s.ShouldReceiveNotification(assigneeRole) {
		assigneeNotifications = "Notifications for assignee: on"
	}

	if s != nil && s.ShouldReceiveNotification(mentionRole) {
		mentionNotifications = "Notifications for mention: on"
	}

	if s != nil && s.ShouldReceiveNotification(reporterRole) {
		reporterNotifications = "Notifications for reporter: on"
	}

	if s != nil && s.ShouldReceiveNotification(watchingRole) {
		watchingNotifications = "Notifications for watching: on"
	}

	return fmt.Sprintf("\t- %s \n\t- %s \n\t- %s \n\t- %s", assigneeNotifications, mentionNotifications, reporterNotifications, watchingNotifications)
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
	connectable, err := p.ensureConnectable(instance.GetID(), types.ID(mattermostUserID))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	if !connectable {
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
	connectable, err := p.ensureConnectable(instanceID, types.ID(mattermostUserID))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	if !connectable {
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

func (p *Plugin) UpdateUserDefaults(mattermostUserID, instanceID types.ID, savedValues *SavedFieldValues) {
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

	if savedValues != nil {
		connection.SavedFieldValues = savedValues
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
	user, instanceID, err := p.ResolveUserInstanceURL(mattermostUserID, instanceURL)
	if err != nil {
		return nil, err
	}

	// The instance may have been uninstalled while the user's record still
	// references it. Clean up the record anyway rather than locking the
	// user out of the one command that can fix it.
	if _, err := p.instanceStore.LoadInstance(instanceID); err != nil {
		if errors.Cause(err) != kvstore.ErrNotFound {
			return nil, err
		}
		p.client.Log.Info("Disconnecting user from an instance that is no longer installed",
			"mattermostUserID", mattermostUserID, "instanceID", instanceID)
	}

	return p.disconnectUser(instanceID, user)
}

func (p *Plugin) SetDefaultInstance(instanceURL string, mattermostUserID types.ID) error {
	user, instance, err := p.LoadUserInstance(mattermostUserID, instanceURL)
	if err != nil {
		return err
	}

	if !user.ConnectedInstances.Contains(instance.GetID()) {
		return errors.Wrapf(kvstore.ErrNotFound, "user is not connected to %q", instance.GetID())
	}

	user.DefaultInstanceID = instance.GetID()

	if err := p.userStore.StoreUser(user); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) disconnectUser(instanceID types.ID, user *User) (*Connection, error) {
	if !user.ConnectedInstances.Contains(instanceID) {
		return nil, errors.Wrapf(kvstore.ErrNotFound, "user is not connected to %q", instanceID)
	}
	// LoadConnection does not error on a missing row; it returns a non-nil,
	// empty Connection, which is fine here since we only need the ID to prune
	// and (if present) the DisplayName for the caller's response.
	conn, err := p.userStore.LoadConnection(instanceID, user.MattermostUserID)
	if err != nil {
		return nil, err
	}

	if user.DefaultInstanceID == instanceID {
		user.DefaultInstanceID = ""
	}

	user.ConnectedInstances.Delete(instanceID)

	err = p.userStore.DeleteConnection(instanceID, user.MattermostUserID)
	if err != nil && errors.Cause(err) != kvstore.ErrNotFound {
		return nil, err
	}
	err = p.userStore.StoreUser(user)
	if err != nil {
		return nil, err
	}

	p.cleanupDMSubscriptionsOnDisconnect(instanceID, user.MattermostUserID.String())

	info, err := p.GetUserInfo(user.MattermostUserID, user)
	if err != nil {
		return nil, err
	}
	// GetUserInfo may have found other stale instances in the record while
	// it was in hand; clean those up too, so a single disconnect recovers a
	// record with more than one dangling instance reference.
	if err := p.healUserRecord(info); err != nil {
		return nil, err
	}

	p.client.Frontend.PublishWebSocketEvent(websocketEventDisconnect, info.AsConfigMap(),
		&model.WebsocketBroadcast{UserId: user.MattermostUserID.String()})

	p.TrackUserEvent("userDisconnected", user.MattermostUserID.String(), nil)

	return conn, nil
}

// reconcileUserInstances drops instances that are no longer installed from
// the user's record, and clears a default that points at one of them. It
// returns the dropped instance IDs, and reports whether it changed the
// record at all, so callers know both what to clean up and whether the
// record needs persisting.
func reconcileUserInstances(user *User, instances *Instances) (dropped []types.ID, changed bool) {
	for _, instanceID := range user.ConnectedInstances.IDs() {
		if !instances.Contains(instanceID) {
			user.ConnectedInstances.Delete(instanceID)
			dropped = append(dropped, instanceID)
		}
	}
	changed = len(dropped) > 0
	if user.DefaultInstanceID != "" && !user.ConnectedInstances.Contains(user.DefaultInstanceID) {
		user.DefaultInstanceID = ""
		changed = true
	}
	return dropped, changed
}

// ensureConnectable reports whether mattermostUserID may start a connection
// flow for instanceID, which the caller must have already confirmed is
// installed. Only the user's own record decides that. A connection row the
// record does not back is orphaned -- typically left behind by a
// since-removed instance that occupied the same URL -- and is cleared here
// so it cannot block the reconnect.
func (p *Plugin) ensureConnectable(instanceID, mattermostUserID types.ID) (bool, error) {
	user, err := p.userStore.LoadUser(mattermostUserID)
	if err != nil {
		if errors.Cause(err) != kvstore.ErrNotFound {
			return false, err
		}
		user = NewUser(mattermostUserID)
	}
	if user.ConnectedInstances.Contains(instanceID) {
		return false, nil
	}

	p.deleteOrphanedConnection(instanceID, mattermostUserID)
	return true, nil
}

// deleteOrphanedConnection removes the connection row, and the Jira account
// reverse index it owns, for an instance the user's record does not list as
// connected. Nothing can legitimately reach the row at this point, so a
// failure to delete it is logged rather than returned.
func (p *Plugin) deleteOrphanedConnection(instanceID, mattermostUserID types.ID) {
	conn, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil || len(conn.JiraAccountID()) == 0 {
		return
	}

	if err := p.userStore.DeleteConnection(instanceID, mattermostUserID); err != nil {
		p.client.Log.Warn("Failed to delete orphaned Jira connection",
			"mattermostUserID", mattermostUserID, "instanceID", instanceID, "error", err.Error())
	}
}

func (p *Plugin) GetJiraUserFromMentions(instanceID types.ID, mentions model.UserMentionMap, userKey string) (*jira.User, error) {
	userKey = strings.TrimPrefix(userKey, "@")
	mentionUser, found := mentions[userKey]
	if !found {
		return nil, errors.New("the mentioned user was not found")
	}

	connection, err := p.userStore.LoadConnection(instanceID, types.ID(mentionUser))
	if err != nil {
		p.client.Log.Warn("Error occurred while loading connection", "User", mentionUser, "Error", err.Error())
		return nil, errors.New("the mentioned user is not connected to Jira")
	}

	if connection.AccountID != "" {
		return &connection.User, nil
	}

	return nil, errors.New("the mentioned user is not connected to Jira")
}
