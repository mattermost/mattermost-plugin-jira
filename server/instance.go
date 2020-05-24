// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type InstanceType string

const (
	CloudInstanceType  = InstanceType("cloud")
	ServerInstanceType = InstanceType("server")
)

type Instance interface {
	GetClient(*Connection) (Client, error)
	GetDisplayDetails() map[string]string
	GetUserConnectURL(mattermostUserId string) (string, error)
	GetManageAppsURL() string
	GetManageWebhooksURL() string
	GetURL() string

	Common() *InstanceCommon
	types.Value
}

// InstanceCommon contains metadata common for both cloud and server instances.
// The fields lack `json` modifiers to be backwards compatible with v2.
type InstanceCommon struct {
	*Plugin       `json:"-"`
	PluginVersion string `json:",omitempty"`

	InstanceID types.ID
	Type       InstanceType
	IsV2Legacy bool
}

func newInstanceCommon(p *Plugin, instanceType InstanceType, instanceID types.ID) *InstanceCommon {
	return &InstanceCommon{
		Plugin:        p,
		Type:          instanceType,
		InstanceID:    instanceID,
		PluginVersion: manifest.Version,
	}
}

func (ic InstanceCommon) AsConfigMap() map[string]interface{} {
	return map[string]interface{}{
		"type":        string(ic.Type),
		"instance_id": string(ic.InstanceID),
	}
}

func (common InstanceCommon) GetID() types.ID {
	return common.InstanceID
}

func (common *InstanceCommon) Common() *InstanceCommon {
	return common
}
