// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"regexp"
	"sync"

	"github.com/andygrunwald/go-jira"
)

const (
	JIRATypeCloud  = "cloud"
	JIRATypeServer = "server"
)

const prefixForInstance = true

const wSEventInstanceStatus = "instance_status"

type Instance interface {
	GetJIRAClient(jiraUser JIRAUser) (*jira.Client, error)
	GetDisplayDetails() map[string]string
	GetMattermostKey() string
	GetPlugin() *Plugin
	GetType() string
	GetURL() string
	GetUserConnectURL(mattermostUserId string) (string, error)
	GetUserGroups(jiraUser JIRAUser) ([]*jira.UserGroup, error)
	Init(p *Plugin)
}

type JIRAInstance struct {
	*Plugin `json:"-"`
	lock    *sync.RWMutex

	Key           string
	Type          string
	PluginVersion string
}

type InstanceStatus struct {
	InstanceInstalled string `json:"instance_installed"`
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func NewJIRAInstance(p *Plugin, typ, key string) *JIRAInstance {
	return &JIRAInstance{
		Plugin:        p,
		Type:          typ,
		Key:           key,
		PluginVersion: manifest.Version,
		lock:          &sync.RWMutex{},
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

func (ji *JIRAInstance) Init(p *Plugin) {
	ji.Plugin = p
	ji.lock = &sync.RWMutex{}
}

type withInstanceFunc func(ji Instance, w http.ResponseWriter, r *http.Request) (int, error)

func withInstance(store CurrentInstanceStore, w http.ResponseWriter, r *http.Request, f withInstanceFunc) (int, error) {
	ji, err := store.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return f(ji, w, r)
}
