// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// TestExecuteConnect_StaleConnectionRow reproduces the scenario where a
// user's own record does not list an installed instance as connected (so
// it is offered by /jira connect), but a connection row for that instance
// still exists in KV -- e.g. left behind by a different, since-removed
// instance that reused the same URL. That row must not block the
// reconnect.
func TestExecuteConnect_StaleConnectionRow(t *testing.T) {
	p := &Plugin{}
	p.updateConfig(func(conf *config) {
		conf.mattermostSiteURL = mattermostSiteURL
	})

	api := &plugintest.API{}
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	testUserID := types.ID("stale-row-user")
	store := getMockUserStoreKV()
	store.users[testUserID] = NewUser(testUserID) // not connected, per this user's own record
	store.connections[testUserID] = &Connection{User: jira.User{AccountID: "stale-account"}}

	var capturedMessage string
	api.On("SendEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Run(func(args mock.Arguments) {
		capturedMessage = args.Get(1).(*model.Post).Message
	}).Once().Return(&model.Post{})

	p.SetAPI(api)
	p.client = pluginapi.NewClient(p.API, p.Driver)
	p.instanceStore = p.getMockInstanceStoreKV(1) // testInstance1 only
	p.userStore = store

	commandArgs := &model.CommandArgs{Command: "/jira connect", UserId: testUserID.String()}
	_, err := p.ExecuteCommand(&plugin.Context{}, commandArgs)
	require.Nil(t, err)

	assert.NotContains(t, capturedMessage, "already have a Jira account linked",
		"a connection row not backed by the user's own ConnectedInstances must not block reconnect")
	assert.Contains(t, capturedMessage, "Click here to link your Jira account")

	_, stillExists := store.connections[testUserID]
	assert.False(t, stillExists, "the stale connection row must be cleared so the new connection isn't corrupted by old data")
}
