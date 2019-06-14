// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package instance

import (
	"regexp"

	"github.com/andygrunwald/go-jira"
)

const (
	InstanceTypeCloud  = "cloud"
	InstanceTypeServer = "server"
)

const prefixForInstance = true

const wSEventInstanceStatus = "instance_status"

type Instance interface {
	GetDisplayDetails() map[string]string
	GetMattermostKey() string
	GetType() string
	GetURL() string
	GetUserConnectURL(Config, SecretStore, string) (string, error)
	GetClient(Config, SecretStore, *JiraUser) (*jira.Client, error)
}

type instance struct {
	Key  string
	Type string
}

type InstanceStatus struct {
	InstanceInstalled bool `json:"instance_installed"`
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func newInstance(typ, key string) *instance {
	return &instance{
		Type: typ,
		Key:  key,
	}
}

func (instance instance) GetKey() string {
	return instance.Key
}

func (instance instance) GetType() string {
	return instance.Type
}
