// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

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
		"multiple projects selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES", "OTHER"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
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
		"CLOUD - custom field selected": {
			WebhookTestData: "webhook-cloud-issue-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10072"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"SERVER - custom field selected": {
			WebhookTestData: "webhook-server-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_numfield"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"CLOUD - custom field selected, wrong field": {
			WebhookTestData: "webhook-cloud-issue-updated-custom-field.json",
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
		"SERVER - custom field selected, wrong field": {
			WebhookTestData: "webhook-server-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_numfield2"),
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
			WebhookTestData: "webhook-cloud-issue-updated-custom-field.json",
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
		"status field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10004")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"status field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10005")},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"status field filter configured to exclude, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10004"), Exclude: true},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"status field filter configured to exclude, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10005"), Exclude: true},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"custom multi-select field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10033")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"custom multi-select field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10001")},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"custom single-select field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10076", Values: NewStringSet("10039")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"custom single-select field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10076", Values: NewStringSet("10001")},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"custom string field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10078", Values: NewStringSet("some value")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"custom string field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10078", Values: NewStringSet("wrong value")},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"custom string array field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "labels", Values: NewStringSet("Label1")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		// we need to explain to the user that if they want to "and" the labels, they need two separate filter rows
		"two filters, custom string array field filter with multiple values configured, one matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "labels", Values: NewStringSet("Label1", "Label3")},
							{Key: "labels", Values: NewStringSet("Label4")},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"custom string array field filter with multiple values configured, one matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "labels", Values: NewStringSet("Label1", "Label3")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"custom string array field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "labels", Values: NewStringSet("wrong value")},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"fixVersions filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "fixVersions", Values: NewStringSet("10000")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"priority filter configured, matches": {
			WebhookTestData: "webhook-server-issue-updated-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("HEY"),
						IssueTypes: NewStringSet("10001"),
						Fields: []FieldFilter{
							{Key: "Priority", Values: NewStringSet("1")},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"custom string field filter configured, field is not present in issue metadata": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "labels2", Values: NewStringSet("some value")},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"custom string field filter configured, field is null in issue metadata": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10026", Values: NewStringSet("some value")},
						},
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
			p.currentInstanceStore = mockCurrentInstanceStore{p}

			subscriptionBytes, err := json.Marshal(tc.Subs)
			assert.Nil(t, err)

			subKey := keyWithMockInstance(JIRA_SUBSCRIPTIONS_KEY)
			api.On("KVGet", subKey).Return(subscriptionBytes, nil)

			api.On("KVCompareAndSet", subKey, subscriptionBytes, mock.MatchedBy(func(data []byte) bool {
				return true
			})).Return(nil)

			data, err := getJiraTestData(tc.WebhookTestData)
			assert.Nil(t, err)

			r := bytes.NewReader(data)
			bb, err := ioutil.ReadAll(r)
			require.Nil(t, err)

			wh, err := ParseWebhook(bb)
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
