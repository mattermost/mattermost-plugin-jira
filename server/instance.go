// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"fmt"
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
	GetJIRAClient(info JIRAUserInfo) (*jira.Client, error)
	GetKey() string
	GetType() string
	GetURL() string
	GetUserConnectURL(p *Plugin, mattermostUserId string) (string, error)
	WrapDatabaseKey(key string) string
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

func (ji JIRAInstance) WrapDatabaseKey(key string) string {
	if prefixForInstance {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", ji.Key, key)
		key = fmt.Sprintf("%x", h.Sum(nil))
	}
	return key
}

func (ji JIRAInstance) GetKey() string {
	return ji.Key
}

func (ji JIRAInstance) GetType() string {
	return ji.Type
}
