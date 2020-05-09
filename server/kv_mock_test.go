// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"fmt"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
)

type jiraTestInstance struct {
	InstanceCommon
}

var _ Instance = (*jiraTestInstance)(nil)

const (
	mockCurrentInstanceURL = "http://jiraTestInstanceURL.some"
)

func keyWithMockInstance(key string) string {
	h := md5.New()
	fmt.Fprintf(h, "%s/%s", mockCurrentInstanceURL, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (jti jiraTestInstance) GetURL() string {
	return mockCurrentInstanceURL
}
func (jti jiraTestInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/apps/manage", mockCurrentInstanceURL)
}
func (jti jiraTestInstance) GetPlugin() *Plugin {
	return jti.Plugin
}
func (jti jiraTestInstance) GetMattermostKey() string {
	return "jiraTestInstanceMattermostKey"
}
func (jti jiraTestInstance) GetDisplayDetails() map[string]string {
	return map[string]string{}
}
func (jti jiraTestInstance) GetUserConnectURL(mattermostUserId string) (string, error) {
	return "http://jiraTestInstanceUserConnectURL.some", nil
}
func (jti jiraTestInstance) GetClient(*Connection) (Client, error) {
	return testClient{}, nil
}
func (jti jiraTestInstance) GetUserGroups(*Connection) ([]*jira.UserGroup, error) {
	return nil, errors.New("not implemented")
}

type mockUserStore struct{}

func (store mockUserStore) StoreUser(*User) error {
	return nil
}
func (store mockUserStore) LoadUser(string) (*User, error) {
	return &User{}, nil
}
func (store mockUserStore) StoreConnection(Instance, string, *Connection) error {
	return nil
}
func (store mockUserStore) LoadConnection(Instance, string) (*Connection, error) {
	return &Connection{}, nil
}
func (store mockUserStore) LoadMattermostUserId(instance Instance, jiraUserName string) (string, error) {
	return "testMattermostUserId012345", nil
}
func (store mockUserStore) DeleteConnection(instance Instance, mattermostUserId string) error {
	return nil
}
func (store mockUserStore) CountUsers() (int, error) {
	return 0, nil
}
