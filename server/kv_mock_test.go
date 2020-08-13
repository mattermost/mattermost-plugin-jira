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
	mockInstance1URL = "jiraurl1"
	mockInstance2URL = "jiraurl2"
)

var testInstance1 = &testInstance{
	InstanceCommon: InstanceCommon{
		InstanceID: mockInstance1URL,
		IsV2Legacy: true,
		Type:       "testInstanceType",
	},
}

var testInstance2 = &testInstance{
	InstanceCommon: InstanceCommon{
		InstanceID: mockInstance2URL,
		Type:       "testInstanceType",
	},
}

func (ti testInstance) GetURL() string {
	return ti.InstanceID.String()
}
func (ti testInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/apps/manage", ti.InstanceID)
}
func (ti testInstance) GetManageWebhooksURL() string {
	return fmt.Sprintf("%s/webhooks/manage", ti.InstanceID)
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
	return fmt.Sprintf("%s/UserConnectURL.some", ti.GetURL()), nil
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
func (store mockUserStore) LoadUser(id types.ID) (*User, error) {
	return NewUser(id), nil
}
func (store mockUserStore) StoreConnection(types.ID, types.ID, *Connection) error {
	return nil
}
func (store mockUserStore) LoadConnection(types.ID, types.ID) (*Connection, error) {
	return &Connection{}, nil
}
func (store mockUserStore) LoadMattermostUserId(instanceID types.ID, jiraUserName string) (types.ID, error) {
	return "testMattermostUserId012345", nil
}
func (store mockUserStore) DeleteConnection(instanceID, mattermostUserID types.ID) error {
	return nil
}
func (store mockUserStore) CountUsers() (int, error) {
	return 0, nil
}
func (store mockUserStore) MapUsers(func(*User) error) error {
	return nil
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
func (store mockInstanceStore) LoadInstanceFullKey(string) (Instance, error) {
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
