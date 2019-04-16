// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"
)

const (
	JIRACloudType  = "cloud"
	JIRAServerType = "server"
)

const prefixForInstance = true

type JIRAInstance interface {
	GetJIRAClientForServer() (*jira.Client, error)
	GetJIRAClientForUser(info JIRAUserInfo) (*jira.Client, *http.Client, error)
	ParseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error)
	GetKey() string
	GetType() string
	GetURL() string
	GetUserConnectURL(p *Plugin, mattermostUserId string) (string, error)
	WrapDatabaseKey(key string) string
}

type jiraInstance struct {
	*Plugin `json:"-"`

	Key  string
	Type string
}

func (ji jiraInstance) WrapDatabaseKey(key string) string {
	if prefixForInstance {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", ji.Key, key)
		key = fmt.Sprintf("%x", h.Sum(nil))
	}
	return key
}

func (ji jiraInstance) GetKey() string {
	return ji.Key
}

func (ji jiraInstance) GetType() string {
	return ji.Type
}
