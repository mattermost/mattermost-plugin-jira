// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
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
		WebhookTestData string
		Subs            *Subscriptions
		ChannelIds      []string
	}{
		"no filters selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet(),
						Projects:   NewStringSet(),
						IssueTypes: NewStringSet(),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"fields match": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"project does not match": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("NOPE"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"no project selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet(),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"issue type does not match": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10002"),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"no issue type selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet(),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"event type does not match": {
			WebhookTestData: "webhook-issue-deleted.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_summary"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"updated all selected": {
			WebhookTestData: "webhook-issue-updated-labels.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"updated all selected, wrong incoming event": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"custom field selected": {
			WebhookTestData: "webhook-issue-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10001"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"custom field selected, wrong field": {
			WebhookTestData: "webhook-issue-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10002"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{},
		},
		"custom field selected is the second of two custom fields in webhook": {
			WebhookTestData: "webhook-issue-updated-multiple-custom-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10002"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"updated all selected, custom field": {
			WebhookTestData: "webhook-issue-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"multiple subscriptions, both acceptable": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId1", "sampleChannelId2"},
		},
		"multiple subscriptions, one acceptable": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_deleted"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId1"},
		},
		"multiple subscriptions, neither acceptable": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_deleted"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_deleted"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
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

			data, err := getJiraTestData(tc.WebhookTestData)
			assert.Nil(t, err)

			r := bytes.NewReader(data)
			wh, err := ParseWebhook(r)
			assert.Nil(t, err)

			actual, err := p.getChannelsSubscribed(wh.(*webhook))
			assert.Nil(t, err)

			assert.Equal(t, len(tc.ChannelIds), len(actual))
			for _, channelId := range tc.ChannelIds {
				assert.Contains(t, actual, channelId)
			}
		})
	}
}
