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

type Instance interface {
	GetDisplayDetails() map[string]string
	GetMattermostKey() string
	GetType() string
	GetURL() string
	GetUserConnectURL(a *Action) (string, error)
	GetJIRAClient(a *Action, jiraUser *JIRAUser) (*jira.Client, error)
}

type JIRAInstance struct {
	Key  string
	Type string
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func NewJIRAInstance(typ, key string) *JIRAInstance {
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
