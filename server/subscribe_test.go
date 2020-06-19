// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

func TestListChannelSubscriptions(t *testing.T) {
	p := &Plugin{}
	p.updateConfig(func(conf *config) {
		conf.Secret = "somesecret"
	})
	p.instanceStore = getMockInstanceStoreKV(false)

	for name, tc := range map[string]struct {
		Subs          *Subscriptions
		RunAssertions func(t *testing.T, actual string)
	}{
		"one subscription": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel1",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				expected := "The following channels have subscribed to Jira notifications. To modify a subscription, navigate to the channel and type `/jira subscribe`\n\n#### Team 1 Display Name\n* **~channel-1-name** (1):\n  * PROJ - Sub Name X"
				assert.Equal(t, expected, actual)
			},
		},
		"zero subscriptions": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{}),
			RunAssertions: func(t *testing.T, actual string) {
				expected := "There are currently no channels subcriptions to Jira notifications. To add a subscription, navigate to a channel and type `/jira subscribe`\n"
				assert.Equal(t, expected, actual)
			},
		},
		"one subscription in DM channel": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel2",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				expected := "The following channels have subscribed to Jira notifications. To modify a subscription, navigate to the channel and type `/jira subscribe`\n\n#### Group and Direct Messages\n* **channel-2-name-DM** (1):\n  * PROJ - Sub Name X"
				assert.Equal(t, expected, actual)
			},
		},
		"one channel with three subscriptions": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel1",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel1",
					Name:      "Sub Name Y",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel1",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				numlines := strings.Count(actual, "\n") + 1
				assert.Equal(t, 7, numlines)
				assert.NotContains(t, actual, "\n#### Group and Direct Messages")
				assert.Contains(t, actual, "\n#### Team 1 Display Name")
				assert.Contains(t, actual, `**~channel-1-name** (3):`)
				assert.Contains(t, actual, `* PROJ - Sub Name X`)
				assert.Contains(t, actual, `* EXT - Sub Name Y`)
				assert.Contains(t, actual, `* EXT - (No Name)`)
			},
		},
		"two channels with multiple subscriptions": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel1",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel1",
					Name:      "Sub Name Y",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "channel2",
					Name:      "Sub Name Z",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				numlines := strings.Count(actual, "\n") + 1
				assert.Equal(t, 10, numlines)
				assert.Contains(t, actual, "\n#### Group and Direct Messages")
				assert.Contains(t, actual, "\n#### Team 1 Display Name")
				assert.Contains(t, actual, `Group and Direct Messages`)
				assert.Contains(t, actual, `**~channel-1-name** (2):`)
				assert.Contains(t, actual, `* PROJ - Sub Name X`)
				assert.Contains(t, actual, `* EXT - Sub Name Y`)
				assert.Contains(t, actual, `**channel-2-name-DM** (1):`)
				assert.Contains(t, actual, `* EXT - Sub Name Z`)
			},
		},
		"two teams with two channels with multiple subscriptions": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        "SubID1a",
					ChannelId: "channel1",
					Name:      "Sub Name 1",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
				},
				ChannelSubscription{
					Id:        "SubID2",
					ChannelId: "channel2",
					Name:      "Sub Name 2",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
				ChannelSubscription{
					Id:        "SubID3",
					ChannelId: "channel3",
					Name:      "Sub Name 3",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
				ChannelSubscription{
					Id:        "SubID4",
					ChannelId: "channel4",
					Name:      "Sub Name 4",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				// test that team names are ordered and channels per team are also shown
				re := regexp.MustCompile("(?s)(.*)Team 1.*channel-1.*Team 2.*channel-(3|4).*channel-(3|4).*Group and Direct Messages.*channel-2")
				assert.Regexp(t, re, actual)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			p.updateConfig(func(conf *config) {
				conf.Secret = "somesecret"
			})
			p.SetAPI(api)

			subscriptionBytes, err := json.Marshal(tc.Subs)
			assert.Nil(t, err)

			api.On("KVGet", testSubKey).Return(subscriptionBytes, nil)

			channel1 := &model.Channel{
				Id:          "channel1",
				TeamId:      "team1Id",
				Name:        "channel-1-name",
				DisplayName: "Channel 1 Display Name",
			}
			api.On("GetChannel", "channel1").Return(channel1, nil)

			channel2 := &model.Channel{
				Id:          "channel2",
				TeamId:      "",
				Name:        "channel-2-name-DM",
				DisplayName: "Channel 2 Display Name",
			}

			channel3 := &model.Channel{
				Id:          "channel3",
				TeamId:      "team2Id",
				Name:        "channel-3-name",
				DisplayName: "Channel 3 Display Name",
			}

			channel4 := &model.Channel{
				Id:          "channel4",
				TeamId:      "team2Id",
				Name:        "channel-4",
				DisplayName: "Channel 4 Display Name",
			}

			api.On("GetChannel", "channel1").Return(channel1, nil)
			api.On("GetChannel", "channel2").Return(channel2, nil)
			api.On("GetChannel", "channel3").Return(channel3, nil)
			api.On("GetChannel", "channel4").Return(channel4, nil)

			team1 := &model.Team{
				Id:          "team1Id",
				Name:        "team-1-name",
				DisplayName: "Team 1 Display Name",
			}
			api.On("GetTeam", "team1Id").Return(team1, nil)

			team2 := &model.Team{
				Id:          "team2Id",
				Name:        "team-2-name",
				DisplayName: "Team 2 Display Name",
			}
			api.On("GetTeam", "team2Id").Return(team2, nil)

			api.On("KVCompareAndSet", testSubKey, subscriptionBytes, mock.MatchedBy(func(data []byte) bool {
				return true
			})).Return(nil)

			actual, err := p.listChannelSubscriptions(testInstance1.InstanceID, team1.Id)
			assert.Nil(t, err)
			assert.NotNil(t, actual)

			tc.RunAssertions(t, actual)
		})
	}
}

func TestGetChannelsSubscribed(t *testing.T) {
	p := &Plugin{}
	p.updateConfig(func(conf *config) {
		conf.Secret = "somesecret"
	})
	p.instanceStore = getMockInstanceStoreKV(false)

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
			ChannelIds: []string{"sampleChannelId"},
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
			ChannelIds: []string{"sampleChannelId"},
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
		"CLOUD - components selected": {
			WebhookTestData: "webhook-issue-updated-components.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        model.NewId(),
					ChannelId: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{{
							Key:       "components",
							Values:    NewStringSet("10000"),
							Inclusion: "include_any",
						}},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
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
							{Key: "status", Values: NewStringSet("10004"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "status", Values: NewStringSet("10005"), Inclusion: FILTER_INCLUDE_ANY},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"status field filter configured to include all values, all are present": {
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
							{Key: "customfield_10068", Values: NewStringSet("10033", "10034"), Inclusion: FILTER_INCLUDE_ALL},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"field filter configured to include all values, one is missing": {
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
							{Key: "customfield_10068", Values: NewStringSet("10033", "10035"), Inclusion: FILTER_INCLUDE_ALL},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"field filter configured to exclude, field is present": {
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
							{Key: "status", Values: NewStringSet("10004"), Inclusion: FILTER_EXCLUDE_ANY},
						},
					},
				},
			}),
			ChannelIds: []string{},
		},
		"status field filter configured to exclude, field is not present": {
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
							{Key: "status", Values: NewStringSet("10005"), Inclusion: FILTER_EXCLUDE_ANY},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"filter configured to empty, field is not present": {
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
							{Key: "customfield_10060", Values: NewStringSet(), Inclusion: FILTER_EMPTY},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
		"filter configured to empty, field is present": {
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
							{Key: "status", Values: NewStringSet(), Inclusion: FILTER_EMPTY},
						},
					},
				},
			}),
			ChannelIds: []string{},
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
							{Key: "customfield_10068", Values: NewStringSet("10033"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10068", Values: NewStringSet("10001"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10076", Values: NewStringSet("10039"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10076", Values: NewStringSet("10001"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10078", Values: NewStringSet("some value"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10078", Values: NewStringSet("wrong value"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10071", Values: NewStringSet("value1"), Inclusion: FILTER_INCLUDE_ANY},
						},
					},
				},
			}),
			ChannelIds: []string{"sampleChannelId"},
		},
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
							{Key: "customfield_10071", Values: NewStringSet("value1", "value3"), Inclusion: FILTER_INCLUDE_ANY},
							{Key: "customfield_10071", Values: NewStringSet("value4"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10071", Values: NewStringSet("value1", "value3"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10071", Values: NewStringSet("wrong value"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "fixVersions", Values: NewStringSet("10000"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "Priority", Values: NewStringSet("1"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "labels2", Values: NewStringSet("some value"), Inclusion: FILTER_INCLUDE_ANY},
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
							{Key: "customfield_10026", Values: NewStringSet("some value"), Inclusion: FILTER_INCLUDE_ANY},
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

			subscriptionBytes, err := json.Marshal(tc.Subs)
			assert.Nil(t, err)

			api.On("KVGet", testSubKey).Return(subscriptionBytes, nil)

			api.On("KVCompareAndSet", testSubKey, subscriptionBytes, mock.MatchedBy(func(data []byte) bool {
				return true
			})).Return(nil)

			data, err := getJiraTestData(tc.WebhookTestData)
			assert.Nil(t, err)

			r := bytes.NewReader(data)
			bb, err := ioutil.ReadAll(r)
			require.Nil(t, err)

			wh, err := ParseWebhook(bb)
			assert.Nil(t, err)

			actual, err := p.getChannelsSubscribed(wh.(*webhook), testInstance1.InstanceID)
			assert.Nil(t, err)

			assert.Equal(t, len(tc.ChannelIds), len(actual))
			for _, channelId := range tc.ChannelIds {
				assert.Contains(t, actual, channelId)
			}
		})
	}
}
