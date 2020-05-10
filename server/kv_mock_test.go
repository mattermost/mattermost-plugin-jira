// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/pkg/errors"
)

type testInstance struct {
	InstanceCommon
}

var _ Instance = (*testInstance)(nil)

const (
	mockInstance1URL = "http://jiraTestInstance1URL.some"
	mockInstance2URL = "http://jiraTestInstance2URL.some"
)

var testInstance1 = newTestInstance(nil, mockInstance1URL)
var testInstance2 = newTestInstance(nil, mockInstance2URL)

func newTestInstance(p *Plugin, id types.ID) Instance {
	return &testInstance{
		InstanceCommon: InstanceCommon{
			URL:    id,
			Plugin: p,
			Type:   "testInstanceType",
		},
	}
}

func (ti testInstance) GetURL() string {
	return ti.URL.String()
}
func (ti testInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/apps/manage", ti.URL)
}
func (ti testInstance) GetPlugin() *Plugin {
	return ti.Plugin
}
func (ti testInstance) GetMattermostKey() string {
	return "jiraTestInstanceMattermostKey"
}
func (ti testInstance) GetDisplayDetails() map[string]string {
	return map[string]string{}
}
func (ti testInstance) GetUserConnectURL(mattermostUserId string) (string, error) {
	return fmt.Sprintf("%s/UserConnectURL.some", ti.URL), nil
}
func (ti testInstance) GetClient(*Connection) (Client, error) {
	return testClient{}, nil
}
func (ti testInstance) GetUserGroups(*Connection) ([]*jira.UserGroup, error) {
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

type mockInstanceStore struct{}

func (store mockInstanceStore) CreateInactiveCloudInstance(types.ID) error {
	return nil
}
func (store mockInstanceStore) DeleteInstance(types.ID) error {
	return nil
}
func (store mockInstanceStore) LoadInstance(types.ID) (Instance, error) {
	return &testInstance{}, nil
}
func (store mockInstanceStore) LoadInstances() (*Instances, error) {
	return NewInstances(), nil
}
func (store mockInstanceStore) StoreInstance(instance Instance) error {
	return nil
}
func (store mockInstanceStore) StoreInstances(*Instances) error {
	return nil
}
func (store mockInstanceStore) UpdateInstances(updatef func(instances *Instances) error) error {
	return nil
}
