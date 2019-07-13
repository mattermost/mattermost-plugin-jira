// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

func TestGetChannelsSubscribed(t *testing.T) {
	p := &Plugin{}
	p.updateConfig(func(conf *config) {
		conf.Secret = "somesecret"
	})

	for name, tc := range map[string]struct {
		TestWebhook io.Reader
		Subs        *Subscriptions
		ChannelIds  []string
	}{
		"no filters selected": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{},
						Project:   []string{},
						IssueType: []string{},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"fields match": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"project does not match": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{"NOPE"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"no project selected": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"issue type does not match": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{"TES"},
						IssueType: []string{"10002"},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"no issue type selected": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{"TES"},
						IssueType: []string{},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"event type does not match": {
			TestWebhook: getJiraTestData("webhook-issue-deleted.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_updated_summary"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"updated all selected": {
			TestWebhook: getJiraTestData("webhook-issue-updated-labels.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_updated_all"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"updated all selected, wrong incoming event": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Event:     []string{"event_updated_all"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"multiple subscriptions, both acceptable": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId1", "sampleChannelId2"},
		},
		"multiple subscriptions, one acceptable": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Event:     []string{"event_created"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Event:     []string{"event_deleted"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId1"},
		},
		"multiple subscriptions, neither acceptable": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Event:     []string{"event_deleted"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Event:     []string{"event_deleted"},
						Project:   []string{"TES"},
						IssueType: []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			p.updateConfig(func(conf *config) {
				conf.Secret = "somesecret"
			})
			p.SetAPI(api)

			var existingBytes []byte
			var err error
			existingBytes, err = json.Marshal(tc.Subs)
			assert.Nil(t, err)

			api.On("KVGet", JIRA_SUBSCRIPTIONS_KEY).Return(existingBytes, nil)

			api.On("KVSet", JIRA_SUBSCRIPTIONS_KEY, mock.MatchedBy(func(data []byte) bool {
				return true
			})).Return(nil)

			wh, jwh, err := ParseWebhook(tc.TestWebhook)
			assert.Nil(t, err)

			actual, err := p.getChannelsSubscribed(wh, jwh)
			assert.Nil(t, err)

			assert.Equal(t, len(tc.ChannelIds), len(actual))
			for _, channelId := range tc.ChannelIds {
				assert.Contains(t, actual, channelId)
			}
		})
	}
}
