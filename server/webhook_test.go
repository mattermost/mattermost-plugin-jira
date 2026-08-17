// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"sync"
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/require"
)

func newTestChannelWebhook() *webhook {
	return &webhook{
		JiraWebhook: &JiraWebhook{
			Issue: jira.Issue{ID: "10001", Key: "PROJ-1"},
		},
		headline: "Actor **commented** on PROJ-1",
	}
}

// isDedupClaimOptions matches the KV options a dedup claim must use: an atomic
// write (no old value expected, since the key shouldn't yet exist) with the
// shared webhook dedup TTL.
func isDedupClaimOptions(opts model.PluginKVSetOptions) bool {
	return opts.Atomic && opts.OldValue == nil && opts.ExpireInSeconds == int64(webhookDedupTTL/time.Second)
}

// isDedupReleaseOptions matches the KV options produced by KV.Delete, which is
// implemented as a plain non-atomic write of a nil value.
func isDedupReleaseOptions(opts model.PluginKVSetOptions) bool {
	return !opts.Atomic && opts.ExpireInSeconds == 0
}

// fakeDedupKV emulates the KV store's atomic-claim semantics keyed on the actual
// dedup key, so tests assert on the dedup identity itself rather than on call
// counts. Returns an accessor for the currently held claims.
func fakeDedupKV(api *plugintest.API) func() []string {
	var mu sync.Mutex
	claimed := map[string]bool{}

	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupClaimOptions)).
		Return(func(key string, _ []byte, _ model.PluginKVSetOptions) (bool, *model.AppError) {
			mu.Lock()
			defer mu.Unlock()
			if claimed[key] {
				return false, nil
			}
			claimed[key] = true
			return true, nil
		})

	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupReleaseOptions)).
		Return(func(key string, _ []byte, _ model.PluginKVSetOptions) (bool, *model.AppError) {
			mu.Lock()
			defer mu.Unlock()
			delete(claimed, key)
			return true, nil
		}).Maybe()

	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		keys := make([]string, 0, len(claimed))
		for k := range claimed {
			keys = append(keys, k)
		}
		return keys
	}
}

func newTestPluginWithAPI(api *plugintest.API) *Plugin {
	p := &Plugin{}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	return p
}

func TestPostToChannelDeduplicatesConcurrentDeliveries(t *testing.T) {
	t.Run("concurrent deliveries for the same channel post only once", func(t *testing.T) {
		api := &plugintest.API{}
		claimedKeys := fakeDedupKV(api)
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil).Once()
		api.On("LogDebug", mockAnythingOfTypeBatch("string", 3)...).Return()

		p := newTestPluginWithAPI(api)

		const callers = 25
		var wg sync.WaitGroup
		wg.Add(callers)
		start := make(chan struct{})
		for i := 0; i < callers; i++ {
			go func() {
				defer wg.Done()
				<-start
				_, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "")
				require.NoError(t, err)
			}()
		}
		close(start)
		wg.Wait()

		// One key means every delivery computed the same dedup identity.
		require.Len(t, claimedKeys(), 1)
		api.AssertExpectations(t)
	})

	t.Run("overlapping subscriptions on the same channel still post only once", func(t *testing.T) {
		// Two subscriptions on the same channel produce identical webhook content
		// but different subscription names; the dedup key must ignore the name.
		api := &plugintest.API{}
		claimedKeys := fakeDedupKV(api)
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil).Once()
		api.On("LogDebug", mockAnythingOfTypeBatch("string", 3)...).Return().Once()

		p := newTestPluginWithAPI(api)

		_, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "subscription-a")
		require.NoError(t, err)

		post, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "subscription-b")
		require.NoError(t, err)
		require.Nil(t, post)

		require.Len(t, claimedKeys(), 1)
		api.AssertExpectations(t)
	})

	t.Run("different channels each get their own post", func(t *testing.T) {
		api := &plugintest.API{}
		claimedKeys := fakeDedupKV(api)
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil).Twice()

		p := newTestPluginWithAPI(api)

		_, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "")
		require.NoError(t, err)
		_, _, err = newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-2", "bot-user-id", "")
		require.NoError(t, err)

		// Two distinct keys means the channel ID is part of the dedup identity.
		require.Len(t, claimedKeys(), 2)
		api.AssertExpectations(t)
	})

	t.Run("a failed post releases the claim so the event can be redelivered", func(t *testing.T) {
		api := &plugintest.API{}
		claimedKeys := fakeDedupKV(api)
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).
			Return((*model.Post)(nil), model.NewAppError("CreatePost", "boom", nil, "", 500)).Once()

		p := newTestPluginWithAPI(api)

		_, status, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "")
		require.Error(t, err)
		require.Equal(t, 500, status)
		require.Empty(t, claimedKeys(), "the claim should be released when the post fails")

		// A redelivery of the same event now succeeds instead of being skipped.
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil).Once()

		post, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "")
		require.NoError(t, err)
		require.NotNil(t, post)
		require.Len(t, claimedKeys(), 1)
		api.AssertExpectations(t)
	})
}
