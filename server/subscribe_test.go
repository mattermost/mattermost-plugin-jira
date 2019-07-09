// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
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
		TestWebhook *JiraWebhook
		Subs        *Subscriptions
		ChannelIds  []string
	}{
		"no filters selected": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: map[string][]string{
						"event":      []string{},
						"project":    []string{},
						"issue_type": []string{},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"project matches": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: map[string][]string{
						"event":      []string{},
						"project":    []string{"TES"},
						"issue_type": []string{},
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
					Filters: map[string][]string{
						"event":      []string{},
						"project":    []string{"NOPE"},
						"issue_type": []string{},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"issue type matches": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: map[string][]string{
						"event":      []string{},
						"project":    []string{},
						"issue_type": []string{"10001"},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"issue type does not match": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: map[string][]string{
						"event":      []string{},
						"project":    []string{},
						"issue_type": []string{"10002"},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"event type matches": {
			TestWebhook: getJiraTestData("webhook-issue-created.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: map[string][]string{
						"event":      []string{"event_created"},
						"project":    []string{},
						"issue_type": []string{},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"event type does not match": {
			TestWebhook: getJiraTestData("webhook-issue-deleted.json"),
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: map[string][]string{
						"event":      []string{"event_created"},
						"project":    []string{},
						"issue_type": []string{},
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
					Filters: map[string][]string{
						"event":      []string{"event_created"},
						"project":    []string{},
						"issue_type": []string{},
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: map[string][]string{
						"event":      []string{},
						"project":    []string{},
						"issue_type": []string{},
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
					Filters: map[string][]string{
						"event":      []string{"event_created"},
						"project":    []string{},
						"issue_type": []string{},
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: map[string][]string{
						"event":      []string{"event_deleted"},
						"project":    []string{},
						"issue_type": []string{},
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
					Filters: map[string][]string{
						"event":      []string{"event_deleted"},
						"project":    []string{},
						"issue_type": []string{},
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId2",
					Filters: map[string][]string{
						"event":      []string{"event_deleted"},
						"project":    []string{},
						"issue_type": []string{},
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

			actual, err := p.getChannelsSubscribed(tc.TestWebhook)
			assert.Nil(t, err)

			assert.Equal(t, len(tc.ChannelIds), len(actual))
			for _, channelId := range tc.ChannelIds {
				assert.Contains(t, actual, channelId)
			}
		})
	}
}
