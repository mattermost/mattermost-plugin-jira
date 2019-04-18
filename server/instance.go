// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"sync"

	"github.com/andygrunwald/go-jira"
)

const (
	JIRATypeCloud  = "cloud"
	JIRATypeServer = "server"
)

const prefixForInstance = true

type Instance interface {
	InitWithPlugin(p *Plugin) Instance
	GetJIRAClient(jiraUser JIRAUser) (*jira.Client, error)
	GetKey() string
	GetType() string
	GetURL() string
	GetUserConnectURL(p *Plugin, mattermostUserId string) (string, error)
}

type JIRAInstance struct {
	*Plugin `json:"-"`
	lock    *sync.RWMutex

	Key  string
	Type string
}

func NewJIRAInstance(p *Plugin, typ, key string) *JIRAInstance {
	return &JIRAInstance{
		Plugin: p,
		Type:   typ,
		Key:    key,
		lock:   &sync.RWMutex{},
	}
}

func (ji JIRAInstance) GetKey() string {
	return ji.Key
}

func (ji JIRAInstance) GetType() string {
	return ji.Type
}
