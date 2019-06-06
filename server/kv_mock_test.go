// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
)

type jiraTestInstance struct {
	instance
}

var _ Instance = (*jiraTestInstance)(nil)

func (jti jiraTestInstance) GetURL() string {
	return "http://jiraTestInstanceURL.some"
}
func (jti jiraTestInstance) GetMattermostKey() string {
	return "jiraTestInstanceMattermostKey"
}
func (jti jiraTestInstance) GetDisplayDetails() map[string]string {
	return map[string]string{}
}
func (jti jiraTestInstance) GetUserConnectURL(Config, SecretsStore, string) (string, error) {
	return "http://jiraTestInstanceUserConnectURL.some", nil
}
func (jti jiraTestInstance) GetClient(Config, SecretsStore, *JiraUser) (*jira.Client, error) {
	return nil, errors.New("not implemented")
}

type mockCurrentInstanceStore struct {
	plugin *Plugin
}

func (store mockCurrentInstanceStore) StoreCurrentInstance(instance Instance) error {
	return nil
}
func (store mockCurrentInstanceStore) LoadCurrentInstance() (Instance, error) {
	return &jiraTestInstance{
		instance: *newInstance("test", "jiraTestInstanceKey"),
	}, nil
}

type mockUserStore struct{}

func (store mockUserStore) StoreUserInfo(instance Instance, mattermostUserId string, jiraUser JiraUser) error {
	return nil
}
func (store mockUserStore) LoadJiraUser(instance Instance, mattermostUserId string) (JiraUser, error) {
	return JiraUser{}, nil
}
func (store mockUserStore) LoadMattermostUserId(instance Instance, jiraUserName string) (string, error) {
	return "testMattermostUserId012345", nil
}
func (store mockUserStore) DeleteUserInfo(instance Instance, mattermostUserId string) error {
	return nil
}
