// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type UserInfo struct {
	IsConnected bool       `json:"is_connected"`
	CanConnect  bool       `json:"can_connect"`
	User        *User      `json:"user"`
	Instances   *Instances `json:"instances"`

	connectable *Instances
	// reconciled reports whether GetUserInfo dropped any stale instances
	// from User. Callers that hold a durable copy of the record should
	// persist it via StoreUser when this is true.
	reconciled bool
}

func (p *Plugin) httpGetUserInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	info, err := p.GetUserInfo(types.ID(mattermostUserID), nil)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	if info.reconciled {
		if err := p.userStore.StoreUser(info.User); err != nil {
			p.client.Log.Warn("Failed to persist reconciled user record",
				"mattermostUserID", mattermostUserID, "error", err.Error())
		}
	}

	return respondJSON(w, info.AsConfigMap())
}

func (p *Plugin) GetUserInfo(mattermostUserID types.ID, user *User) (*UserInfo, error) {
	var err error

	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return nil, err
	}

	if user == nil {
		user, err = p.MigrateV2User(mattermostUserID)
		if err != nil {
			return nil, err
		}
	}

	// Drop any instances that are no longer installed before computing
	// anything from the record, so a dangling reference to a removed
	// instance can't make IsConnected/CanConnect report a contradictory
	// state.
	reconciled := reconcileUserInstances(user, instances)

	isConnected := !user.ConnectedInstances.IsEmpty()
	connectable := NewInstances()
	for _, instanceID := range instances.IDs() {
		if !user.ConnectedInstances.Contains(instanceID) {
			connectable.Set(instances.Get(instanceID))
		}
	}

	return &UserInfo{
		CanConnect:  !connectable.IsEmpty(),
		IsConnected: isConnected,
		Instances:   instances,
		User:        user,
		connectable: connectable,
		reconciled:  reconciled,
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
		m["user_info"] = info.User.AsConfigMap()
	}
	return m
}
