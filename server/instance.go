// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type Instance interface {
	GetClient(*Connection) (Client, error)
	GetDisplayDetails() map[string]string
	GetUserConnectURL(mattermostUserId string) (string, error)
	GetManageAppsURL() string
	GetURL() string

	Common() *InstanceCommon
	types.Value
}

type InstanceCommon struct {
	*Plugin       `json:"-"`
	PluginVersion string `json:",omitempty"`

	URL       types.ID
	Alias     string
	Type      string
	IsDefault bool
}

func newInstanceCommon(p *Plugin, typ string, url types.ID) *InstanceCommon {
	return &InstanceCommon{
		Plugin:        p,
		Type:          typ,
		URL:           url,
		PluginVersion: manifest.Version,
	}
}

func (common InstanceCommon) GetID() types.ID {
	return common.URL
}

func (common *InstanceCommon) Common() *InstanceCommon {
	return common
}
