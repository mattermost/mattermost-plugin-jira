// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type testInstance struct {
	InstanceCommon
}

var _ Instance = (*testInstance)(nil)

const (
	mockInstance1URL = "https://jiraurl1.com"
	mockInstance2URL = "https://jiraurl2.com"
	mockInstance3URL = "https://jiraurl3.com"
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
func (ti testInstance) GetJiraBaseURL() string {
	return ti.GetURL()
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
func (ti testInstance) GetUserConnectURL(mattermostUserID string) (string, *http.Cookie, error) {
	return fmt.Sprintf("%s/UserConnectURL.some", ti.GetURL()), nil, nil
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
	return &Connection{
		Settings: &ConnectionSettings{
			Notifications: true,
		},
	}, nil
}
func (store mockUserStore) LoadMattermostUserID(instanceID types.ID, jiraUserName string) (types.ID, error) {
	return mockUserIDWithNotifications, nil
}
func (store mockUserStore) DeleteConnection(instanceID, mattermostUserID types.ID) error {
	return nil
}
func (store mockUserStore) CountUsers() (int, error) {
	return 0, nil
}
func (store mockUserStore) MapUsers(func(*User) error) (int, error) {
	return 0, nil
}

type mockInstanceStore struct {
	mock.Mock
}

func (store *mockInstanceStore) CreateInactiveCloudInstance(types.ID, string) (string, error) {
	return strings.Repeat("a", 64), nil
}
func (store *mockInstanceStore) LoadPendingCloudSetupRoute(types.ID) (types.ID, error) {
	return "", nil
}

func (store *mockInstanceStore) StorePendingCloudSetupRoute(types.ID, types.ID) error {
	return nil
}

func (store *mockInstanceStore) DeletePendingCloudSetupRoute(types.ID) error {
	return nil
}

func (store *mockInstanceStore) DeleteInstance(types.ID) error {
	return nil
}
func (store *mockInstanceStore) LoadInstance(id types.ID) (Instance, error) {
	args := store.Called(id)
	return args.Get(0).(Instance), args.Error(1)
}
func (store *mockInstanceStore) LoadInstanceFullKey(string) (Instance, error) {
	return &testInstance{}, nil
}
func (store *mockInstanceStore) LoadInstances() (*Instances, error) {
	return NewInstances(), nil
}
func (store *mockInstanceStore) StoreInstance(instance Instance) error {
	return nil
}
func (store *mockInstanceStore) StoreInstances(*Instances) error {
	return nil
}

// instanceStoreDouble exists alongside mockInstanceStoreKV because it wraps
// kvstore.ErrNotFound like the real store, and records durable write order.
type instanceStoreDouble struct {
	mockInstanceStore
	instances *Instances
	blobs     map[types.ID]Instance
	writes    []string
}

func newInstanceStoreDouble(installed ...Instance) *instanceStoreDouble {
	s := &instanceStoreDouble{instances: NewInstances(), blobs: map[types.ID]Instance{}}
	for _, instance := range installed {
		s.instances.Set(instance.Common())
		s.blobs[instance.GetID()] = instance
	}
	return s
}

func (s *instanceStoreDouble) LoadInstances() (*Instances, error) {
	return s.instances, nil
}

func (s *instanceStoreDouble) StoreInstances(instances *Instances) error {
	s.instances = instances
	s.writes = append(s.writes, "StoreInstances")
	return nil
}

func (s *instanceStoreDouble) LoadInstance(id types.ID) (Instance, error) {
	instance, ok := s.blobs[id]
	if !ok {
		return nil, errors.Wrap(kvstore.ErrNotFound, string(id))
	}
	return instance, nil
}

func (s *instanceStoreDouble) StoreInstance(instance Instance) error {
	s.blobs[instance.GetID()] = instance
	return nil
}

func (s *instanceStoreDouble) DeleteInstance(id types.ID) error {
	delete(s.blobs, id)
	s.writes = append(s.writes, "DeleteInstance")
	return nil
}

type connKey struct {
	instanceID       types.ID
	mattermostUserID types.ID
}

// limboUserStore tracks DeleteConnection calls, and mirrors the real store's
// empty-Connection-not-error behavior for a missing row.
type limboUserStore struct {
	mockUserStore
	users              map[types.ID]*User
	connections        map[connKey]*Connection
	deletedConnections []connKey

	// When positive, fails every StoreUser call past the first that many.
	storeUserErrAfter int
	storeUserCalls    int
}

func (s *limboUserStore) LoadUser(id types.ID) (*User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, errors.Wrapf(kvstore.ErrNotFound, "user %q", id)
	}
	return user, nil
}

func (s *limboUserStore) StoreUser(user *User) error {
	s.storeUserCalls++
	if s.storeUserErrAfter > 0 && s.storeUserCalls > s.storeUserErrAfter {
		return errors.New("TESTING kv store unavailable")
	}
	s.users[user.MattermostUserID] = user
	return nil
}

func (s *limboUserStore) LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error) {
	conn, ok := s.connections[connKey{instanceID, mattermostUserID}]
	if !ok {
		return &Connection{MattermostUserID: mattermostUserID}, nil
	}
	return conn, nil
}

func (s *limboUserStore) DeleteConnection(instanceID, mattermostUserID types.ID) error {
	key := connKey{instanceID, mattermostUserID}
	s.deletedConnections = append(s.deletedConnections, key)
	delete(s.connections, key)
	return nil
}

// Logging calls are variadic and testify matches them by expanded argument
// count, so every arity these paths use has to be registered below.
func newPluginForStoreTests(t *testing.T, instanceStore InstanceStore) *Plugin {
	t.Helper()
	p := &Plugin{}
	api := &plugintest.API{}

	bundlePath, err := filepath.Abs("..")
	require.NoError(t, err)
	api.On("GetBundlePath").Return(bundlePath, nil).Maybe()
	api.On("GetConfig").Return(&model.Config{}).Maybe()
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("RegisterCommand", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	for _, level := range []string{"LogDebug", "LogInfo", "LogWarn", "LogError"} {
		for n := 1; n <= 13; n += 2 {
			api.On(level, mockAnythingBatch(n)...).Maybe()
		}
	}

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.instanceStore = instanceStore
	p.tracker = &mockTelemetryTracker{}
	return p
}

func mockAnythingBatch(n int) []interface{} {
	args := make([]interface{}, n)
	for i := range args {
		args[i] = mock.Anything
	}
	return args
}
