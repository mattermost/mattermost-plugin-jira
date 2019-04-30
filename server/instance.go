// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"sync"

	"github.com/andygrunwald/go-jira"
)

const (
	JIRATypeCloud  = "cloud"
	JIRATypeServer = "server"
)

const prefixForInstance = true

type Instance interface {
	GetJIRAClient(jiraUser JIRAUser) (*jira.Client, error)
	GetKey() string
	GetPlugin() *Plugin
	GetType() string
	GetURL() string
	GetUserConnectURL(mattermostUserId string) (string, error)
	SetPlugin(p *Plugin)
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

func (ji JIRAInstance) GetPlugin() *Plugin {
	return ji.Plugin
}

func (ji *JIRAInstance) SetPlugin(p *Plugin) {
	ji.Plugin = p
}

type withInstanceFunc func(ji Instance, w http.ResponseWriter, r *http.Request) (int, error)

func withInstance(p *Plugin, w http.ResponseWriter, r *http.Request, f withInstanceFunc) (int, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return f(ji, w, r)
}
