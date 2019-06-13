// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package server

import (
	"regexp"

	gojira "github.com/andygrunwald/go-jira"
)

const (
	InstanceTypeCloud  = "cloud"
	InstanceTypeServer = "server"
)

type Instance interface {
	GetDisplayDetails() map[string]string
	GetMattermostKey() string
	GetType() string
	GetURL() string
	GetUserConnectURL(Config, SecretStore, string) (string, error)
	GetClient(Config, SecretStore, *JiraUser) (*gojira.Client, error)
}

type JiraUser struct {
	gojira.User
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
	// TODO why is this a pointer?
	Settings *UserSettings
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

