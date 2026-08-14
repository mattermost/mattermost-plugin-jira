// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"sync"
	"sync/atomic"
	"testing"

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

func TestPostToChannelDeduplicatesConcurrentDeliveries(t *testing.T) {
	t.Run("concurrent deliveries for the same channel post only once", func(t *testing.T) {
		api := &plugintest.API{}

		var kvWinners atomic.Int32
		api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(func(string, []byte, model.PluginKVSetOptions) (bool, *model.AppError) {
			if kvWinners.Add(1) == 1 {
				return true, nil
			}
			return false, nil
		})
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil).Once()

		p := &Plugin{}
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

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

		api.AssertExpectations(t)
	})

	t.Run("overlapping subscriptions on the same channel still post only once", func(t *testing.T) {
		// Two subscriptions on the same channel produce identical webhook content
		// but different subscription names; the dedup key must ignore the name.
		api := &plugintest.API{}
		var kvWinners atomic.Int32
		api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(func(string, []byte, model.PluginKVSetOptions) (bool, *model.AppError) {
			if kvWinners.Add(1) == 1 {
				return true, nil
			}
			return false, nil
		}).Twice()
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil).Once()

		p := &Plugin{}
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		_, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "subscription-a")
		require.NoError(t, err)

		post, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "subscription-b")
		require.NoError(t, err)
		require.Nil(t, post)

		api.AssertExpectations(t)
	})

	t.Run("different channels each get their own post", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, (*model.AppError)(nil)).Twice()
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{Id: "post1"}, nil).Twice()

		p := &Plugin{}
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		_, _, err := newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-1", "bot-user-id", "")
		require.NoError(t, err)
		_, _, err = newTestChannelWebhook().PostToChannel(p, "instance-1", "channel-2", "bot-user-id", "")
		require.NoError(t, err)

		api.AssertExpectations(t)
	})
}
