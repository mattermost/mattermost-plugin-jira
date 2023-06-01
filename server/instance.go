// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type InstanceType string

const (
	CloudInstanceType      = InstanceType("cloud")
	ServerInstanceType     = InstanceType("server")
	CloudOAuthInstanceType = InstanceType("cloud-oauth")
)

type Instance interface {
	GetClient(*Connection) (Client, error)
	GetDisplayDetails() map[string]string
	GetUserConnectURL(mattermostUserID string) (string, *http.Cookie, error)
	GetManageAppsURL() string
	GetManageWebhooksURL() string
	GetURL() string
	GetJiraBaseURL() string

	Common() *InstanceCommon
	types.Value
}

// InstanceCommon contains metadata common for both cloud and server instances.
// The fields lack `json` modifiers to be backwards compatible with v2.
type InstanceCommon struct {
	*Plugin       `json:"-"`
	PluginVersion string `json:",omitempty"`

	InstanceID types.ID
	Alias      string
	Type       InstanceType
	IsV2Legacy bool

	SetupWizardUserID string
}

func newInstanceCommon(p *Plugin, instanceType InstanceType, instanceID types.ID) *InstanceCommon {
	return &InstanceCommon{
		Plugin:        p,
		Type:          instanceType,
		InstanceID:    instanceID,
		PluginVersion: Manifest.Version,
	}
}

func (ic InstanceCommon) AsConfigMap() map[string]interface{} {
	return map[string]interface{}{
		"type":        string(ic.Type),
		"instance_id": string(ic.InstanceID),
		"alias":       ic.Alias,
	}
}

func (ic InstanceCommon) GetID() types.ID {
	return ic.InstanceID
}

func (ic *InstanceCommon) Common() *InstanceCommon {
	return ic
}
