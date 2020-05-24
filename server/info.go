// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type UserInfo struct {
	IsConnected            bool       `json:"is_connected"`
	CanConnect             bool       `json:"can_connect"`
	User                   *User      `json:"user"`
	Instances              *Instances `json:"instances"`
	DefaultConnectInstance Instance   `json:"default_connect_instance,omitempty"`
	DefaultUserInstance    Instance   `json:"default_user_instance,omitempty"`
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
	// return respondJSON(w, info)
	return respondJSON(w, info.AsConfigMap())
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
		DefaultUserInstance:    globalDefaultInstance,
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
	if info.DefaultUserInstance != nil {
		m["default_use_instance"] = info.DefaultUserInstance.Common().AsConfigMap()
	}
	return m
}
