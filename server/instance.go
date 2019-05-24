// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
	"regexp"
	"sync"

	"github.com/andygrunwald/go-jira"
)

const (
	JIRATypeCloud         = "cloud"
	JIRATypeServer        = "server"
	wSEventInstanceStatus = "instance_status"
)

const prefixForInstance = true

type Instance interface {
	GetJIRAClient(jiraUser JIRAUser) (*jira.Client, error)
	GetDisplayDetails() map[string]string
	GetMattermostKey() string
	GetPlugin() *Plugin
	GetType() string
	GetURL() string
	GetUserConnectURL(mattermostUserId string) (string, error)
	Init(p *Plugin)
}

type JIRAInstance struct {
	*Plugin `json:"-"`
	lock    *sync.RWMutex

	Key  string
	Type string
}

type InstanceStatus struct {
	InstanceInstalled bool `json:"instance_installed"`
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

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

func (ji *JIRAInstance) Init(p *Plugin) {
	ji.Plugin = p
	ji.lock = &sync.RWMutex{}
}

type withInstanceFunc func(ji Instance, w http.ResponseWriter, r *http.Request) (int, error)

func withInstance(p *Plugin, w http.ResponseWriter, r *http.Request, f withInstanceFunc) (int, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return f(ji, w, r)
}

func httpAPIGetInstanceStatus(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	resp := InstanceStatus{InstanceInstalled: true}

	_, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		resp = InstanceStatus{InstanceInstalled: false}
	}

	b, _ := json.Marshal(resp)
	_, err = w.Write(b)
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}
