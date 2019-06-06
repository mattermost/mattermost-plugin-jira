// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"regexp"

	"github.com/andygrunwald/go-jira"
)

const (
	JIRATypeCloud  = "cloud"
	JIRATypeServer = "server"
)

const prefixForInstance = true

const wSEventInstanceStatus = "instance_status"

type Instance interface {
	GetDisplayDetails() map[string]string
	GetMattermostKey() string
	GetType() string
	GetURL() string
	GetUserConnectURL(Config, SecretsStore, string) (string, error)
	GetJIRAClient(Config, SecretsStore, *JIRAUser) (*jira.Client, error)
}

type JIRAInstance struct {
	Key  string
	Type string
}

type InstanceStatus struct {
	InstanceInstalled bool `json:"instance_installed"`
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func newJIRAInstance(typ, key string) *JIRAInstance {
	return &JIRAInstance{
		Type: typ,
		Key:  key,
	}
}

func (ji JIRAInstance) GetKey() string {
	return ji.Key
}

func (ji JIRAInstance) GetType() string {
	return ji.Type
}
