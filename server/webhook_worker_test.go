// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// TestWebhookWorkerDeliveryGuard covers gating subscription delivery into DM/GM channels
// on a member still being connected, and removing the subscription when none is. The
// "webhook-issue-created.json" fixture skips the notification and watcher code paths,
// leaving the guard as the only thing under test.
func TestWebhookWorkerDeliveryGuard(t *testing.T) {
	const (
		botUserID           = "botuser___________________"
		connectedUserID     = "connecteduser______________"
		disconnectedUserID  = "disconnecteduser___________"
		fixtureIssueID      = "10040"
		fixtureWebhookEvent = "event_created"
	)

	newSub := func(id, channelID string) ChannelSubscription {
		return ChannelSubscription{
			ID:        id,
			ChannelID: channelID,
			Filters: SubscriptionFilters{
				Events: NewStringSet(fixtureWebhookEvent),
			},
		}
	}

	loadWebhookData := func(t *testing.T) []byte {
		data, err := getJiraTestData("webhook-issue-created.json")
		require.NoError(t, err)
		return data
	}

	setup := func(t *testing.T, sub ChannelSubscription) (*plugintest.API, *Plugin) {
		api := &plugintest.API{}
		p := &Plugin{}

		p.updateConfig(func(conf *config) {
			conf.Secret = someSecret
			conf.botUserID = botUserID
			conf.HideDecriptionComment = true
			conf.ThreadedJiraCommentSubscriptionDuration = "30"
		})
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)
		p.instanceStore = p.getMockInstanceStoreKV(1)
		p.userStore = getMockUserStoreKV()

		existing := withExistingChannelSubscriptions([]ChannelSubscription{sub})
		existingBytes, err := json.Marshal(existing)
		require.NoError(t, err)
		api.On("KVGet", testSubKey).Return(existingBytes, nil)

		return api, p
	}

	t.Run("skips DM delivery and self-heals when no member is connected", func(t *testing.T) {
		channelID := "dmchannelaaaaaaaaaaaaaaaa"
		api, p := setup(t, newSub("sub1______________________", channelID))

		api.On("GetChannel", channelID).Return(&model.Channel{Id: channelID, Type: model.ChannelTypeDirect}, nil)
		api.On("GetChannelMembers", channelID, 0, maxDMGMChannelMembers).Return(model.ChannelMembers{
			{UserId: botUserID},
			{UserId: disconnectedUserID},
		}, nil)
		api.On("LogInfo", mockAnythingOfTypeBatch("string", 5)...).Return()
		api.On("KVSetWithOptions", testSubKey, mock.MatchedBy(func(data []byte) bool {
			var savedSubs Subscriptions
			if err := json.Unmarshal(data, &savedSubs); err != nil {
				return false
			}
			return len(savedSubs.Channel.ByID) == 0
		}), mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil)

		ww := webhookWorker{id: 1, p: p}
		err := ww.process(&webhookMessage{InstanceID: testInstance1.GetID(), Data: loadWebhookData(t)})
		require.NoError(t, err)

		api.AssertNotCalled(t, "CreatePost", mock.Anything)
		api.AssertExpectations(t)
	})

	t.Run("delivers to DM when a member is still connected", func(t *testing.T) {
		channelID := "dmchannelbbbbbbbbbbbbbbbb"
		api, p := setup(t, newSub("sub2______________________", channelID))

		p.userStore = mockUserStoreKVWithConnected(types.ID(connectedUserID))

		api.On("GetChannel", channelID).Return(&model.Channel{Id: channelID, Type: model.ChannelTypeDirect}, nil)
		api.On("GetChannelMembers", channelID, 0, maxDMGMChannelMembers).Return(model.ChannelMembers{
			{UserId: botUserID},
			{UserId: connectedUserID},
		}, nil)
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "createdpost1"}, nil)
		api.On("KVSetWithOptions", fmt.Sprintf(ticketRootPostIDKey, fixtureIssueID, channelID), mock.Anything, mock.Anything).Return(true, nil)

		ww := webhookWorker{id: 1, p: p}
		err := ww.process(&webhookMessage{InstanceID: testInstance1.GetID(), Data: loadWebhookData(t)})
		require.NoError(t, err)

		api.AssertCalled(t, "CreatePost", mock.AnythingOfType("*model.Post"))
		api.AssertNotCalled(t, "KVSetWithOptions", testSubKey, mock.Anything, mock.Anything)
	})

	t.Run("delivers to regular channels without checking membership", func(t *testing.T) {
		channelID := "openchannelaaaaaaaaaaaaaa"
		api, p := setup(t, newSub("sub3______________________", channelID))

		api.On("GetChannel", channelID).Return(&model.Channel{Id: channelID, Type: model.ChannelTypeOpen}, nil)
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "createdpost2"}, nil)
		api.On("KVSetWithOptions", fmt.Sprintf(ticketRootPostIDKey, fixtureIssueID, channelID), mock.Anything, mock.Anything).Return(true, nil)

		ww := webhookWorker{id: 1, p: p}
		err := ww.process(&webhookMessage{InstanceID: testInstance1.GetID(), Data: loadWebhookData(t)})
		require.NoError(t, err)

		api.AssertCalled(t, "CreatePost", mock.AnythingOfType("*model.Post"))
		api.AssertNotCalled(t, "GetChannelMembers", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("keeps the subscription when the connection lookup fails", func(t *testing.T) {
		channelID := "dmchannelcccccccccccccccc"
		api, p := setup(t, newSub("sub4______________________", channelID))
		p.userStore = failingConnectionUserStore{UserStore: p.userStore}

		api.On("GetChannel", channelID).Return(&model.Channel{Id: channelID, Type: model.ChannelTypeDirect}, nil)
		api.On("GetChannelMembers", channelID, 0, maxDMGMChannelMembers).Return(model.ChannelMembers{
			{UserId: botUserID},
			{UserId: disconnectedUserID},
		}, nil)
		api.On("LogWarn", mockAnythingOfTypeBatch("string", 7)...).Return()

		ww := webhookWorker{id: 1, p: p}
		err := ww.process(&webhookMessage{InstanceID: testInstance1.GetID(), Data: loadWebhookData(t)})
		require.NoError(t, err)

		api.AssertNotCalled(t, "CreatePost", mock.Anything)
		api.AssertNotCalled(t, "KVSetWithOptions", testSubKey, mock.Anything, mock.Anything)
	})
}

type failingConnectionUserStore struct {
	UserStore
}

func (failingConnectionUserStore) LoadConnection(types.ID, types.ID) (*Connection, error) {
	return nil, errors.New("kv store unavailable")
}
