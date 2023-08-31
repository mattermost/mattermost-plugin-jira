// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

func TestValidateSubscription(t *testing.T) {
	p := &Plugin{}

	p.instanceStore = p.getMockInstanceStoreKV(0)

	api := &plugintest.API{}
	p.SetAPI(api)

	for name, tc := range map[string]struct {
		subscription          *ChannelSubscription
		errorMessage          string
		disableSecurityConfig bool
	}{
		"no event selected": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet(),
					Projects:   NewStringSet("project"),
					IssueTypes: NewStringSet("10001"),
				},
			},
			errorMessage: "please provide at least one event type",
		},
		"no project selected": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet(),
					IssueTypes: NewStringSet("10001"),
				},
			},
			errorMessage: "please provide a project identifier",
		},
		"no issue type selected": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet("project"),
					IssueTypes: NewStringSet(),
				},
			},
			errorMessage: "please provide at least one issue type",
		},
		"valid subscription": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet("project"),
					IssueTypes: NewStringSet("10001"),
				},
			},
			errorMessage: "",
		},
		"valid subscription with security level": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet("TEST"),
					IssueTypes: NewStringSet("10001"),
					Fields: []FieldFilter{
						{
							Key:       "security",
							Inclusion: FilterIncludeAll,
							Values:    NewStringSet("10001"),
						},
					},
				},
			},
			errorMessage: "",
		},
		"invalid 'Exclude' of security level": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet("TEST"),
					IssueTypes: NewStringSet("10001"),
					Fields: []FieldFilter{
						{
							Key:       "security",
							Inclusion: FilterExcludeAny,
							Values:    NewStringSet("10001"),
						},
					},
				},
			},
			errorMessage: "security level does not allow for an \"Exclude\" clause",
		},
		"security config disabled, valid 'Exclude' of security level": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet("TEST"),
					IssueTypes: NewStringSet("10001"),
					Fields: []FieldFilter{
						{
							Key:       "security",
							Inclusion: FilterExcludeAny,
							Values:    NewStringSet("10001"),
						},
					},
				},
			},
			disableSecurityConfig: true,
			errorMessage:          "",
		},
		"invalid access to security level": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet("TEST"),
					IssueTypes: NewStringSet("10001"),
					Fields: []FieldFilter{
						{
							Key:       "security",
							Inclusion: FilterIncludeAll,
							Values:    NewStringSet("10002"),
						},
					},
				},
			},
			errorMessage: "invalid access to security level",
		},
		"user does not have read access to the project": {
			subscription: &ChannelSubscription{
				ID:         "id",
				Name:       "name",
				ChannelID:  "channelid",
				InstanceID: "instance_id",
				Filters: SubscriptionFilters{
					Events:     NewStringSet("issue_created"),
					Projects:   NewStringSet(nonExistantProjectKey),
					IssueTypes: NewStringSet("10001"),
				},
			},
			errorMessage: "failed to get project \"FP\": Project FP not found",
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p.SetAPI(api)
			p.client = pluginapi.NewClient(p.API, p.Driver)

			api.On("KVGet", testSubKey).Return(nil, nil)

			p.updateConfig(func(conf *config) {
				conf.SecurityLevelEmptyForJiraSubscriptions = !tc.disableSecurityConfig
			})

			client := testClient{}
			err := p.validateSubscription(testInstance1.InstanceID, tc.subscription, client)

			if tc.errorMessage == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.errorMessage, err.Error())
			}
		})
	}
}

func TestListChannelSubscriptions(t *testing.T) {
	p := &Plugin{}
	p.updateConfig(func(conf *config) {
		conf.Secret = someSecret
	})
	p.client = pluginapi.NewClient(p.API, p.Driver)
	p.instanceStore = p.getMockInstanceStoreKV(0)

	for name, tc := range map[string]struct {
		Subs          *Subscriptions
		RunAssertions func(t *testing.T, actual string)
	}{
		"one subscription": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        model.NewId(),
					ChannelID: "channel1",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
					InstanceID: testInstance1.GetID(),
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				expected := "The following channels have subscribed to Jira notifications. To modify a subscription, navigate to the channel and type `/jira subscribe edit`\n\n#### Team 1 Display Name\n* **~channel-1-name** (1):\n\t* (1) jiraurl1\n\t\t* PROJ - Sub Name X"
				assert.Equal(t, expected, actual)
			},
		},
		"zero subscriptions": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{}),
			RunAssertions: func(t *testing.T, actual string) {
				expected := "There are currently no channels subcriptions to Jira notifications. To add a subscription, navigate to a channel and type `/jira subscribe edit`\n"
				assert.Equal(t, expected, actual)
			},
		},
		"one subscription in DM channel": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        model.NewId(),
					ChannelID: "channel2",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
					InstanceID: testInstance1.GetID(),
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				expected := "The following channels have subscribed to Jira notifications. To modify a subscription, navigate to the channel and type `/jira subscribe edit`\n\n#### Group and Direct Messages\n* **channel-2-name-DM** (1):\n\t* (1) jiraurl1\n\t\t* PROJ - Sub Name X"
				assert.Equal(t, expected, actual)
			},
		},
		"one channel with three subscriptions": {
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        model.NewId(),
					ChannelID: "channel1",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
					InstanceID: testInstance1.GetID(),
				},
				{
					ID:        model.NewId(),
					ChannelID: "channel1",
					Name:      "Sub Name Y",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
					InstanceID: testInstance1.GetID(),
				},
				{
					ID:        model.NewId(),
					ChannelID: "channel1",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
					InstanceID: testInstance1.GetID(),
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				numlines := strings.Count(actual, "\n") + 1
				assert.Equal(t, 8, numlines)
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
				{
					ID:        model.NewId(),
					ChannelID: "channel1",
					Name:      "Sub Name X",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
				},
				{
					ID:        model.NewId(),
					ChannelID: "channel1",
					Name:      "Sub Name Y",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
				{
					ID:        model.NewId(),
					ChannelID: "channel2",
					Name:      "Sub Name Z",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
			}),
			RunAssertions: func(t *testing.T, actual string) {
				numlines := strings.Count(actual, "\n") + 1
				assert.Equal(t, 12, numlines)
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
				{
					ID:        "SubID1a",
					ChannelID: "channel1",
					Name:      "Sub Name 1",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("PROJ"),
					},
				},
				{
					ID:        "SubID2",
					ChannelID: "channel2",
					Name:      "Sub Name 2",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
				{
					ID:        "SubID3",
					ChannelID: "channel3",
					Name:      "Sub Name 3",
					Filters: SubscriptionFilters{
						Projects: NewStringSet("EXT"),
					},
				},
				{
					ID:        "SubID4",
					ChannelID: "channel4",
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
				conf.Secret = someSecret
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

			p.client = pluginapi.NewClient(api, p.Driver)
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
		conf.Secret = someSecret
	})
	p.instanceStore = p.getMockInstanceStoreKV(0)

	for name, tc := range map[string]struct {
		WebhookTestData       string
		Subs                  *Subscriptions
		ChannelSubscriptions  []ChannelSubscription
		disableSecurityConfig bool
	}{
		"no filters selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet(),
						Projects:   NewStringSet(),
						IssueTypes: NewStringSet(),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"fields match": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"project does not match": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("NOPE"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"no project selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet(),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet(),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"multiple projects selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES", "OTHER"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES", "OTHER"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"issue type does not match": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10002"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"no issue type selected": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet(),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet(),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"event type does not match": {
			WebhookTestData: "webhook-issue-deleted.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_summary"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"updated all selected": {
			WebhookTestData: "webhook-issue-updated-labels.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"updated all selected, wrong incoming event": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"CLOUD - components selected": {
			WebhookTestData: "webhook-issue-updated-components.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
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
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
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
					InstanceID: "jiraurl1",
				},
			},
		},
		"CLOUD - custom field selected": {
			WebhookTestData: "webhook-cloud-issue-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10072"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10072"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"SERVER - custom field selected": {
			WebhookTestData: "webhook-server-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_numfield"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_numfield"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"CLOUD - custom field selected, wrong field": {
			WebhookTestData: "webhook-cloud-issue-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10002"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"SERVER - custom field selected, wrong field": {
			WebhookTestData: "webhook-server-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{

				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_numfield2"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"custom field selected is the second of two custom fields in webhook": {
			WebhookTestData: "webhook-issue-updated-multiple-custom-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10002"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_customfield_10002"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"updated all selected, custom field": {
			WebhookTestData: "webhook-cloud-issue-updated-custom-field.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"multiple subscriptions, both acceptable": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				{
					ID:        "8hduqxgiwiyi5fw3q4d6q56uho",
					ChannelID: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
				{
					ID:        "8hduqxgiwiyi5fw3q4d6q56uho",
					ChannelID: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"multiple subscriptions, one acceptable": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				{
					ID:        "8hduqxgiwiyi5fw3q4d6q56uho",
					ChannelID: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_deleted"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
					InstanceID: "jiraurl1",
				}},
		},
		"multiple subscriptions, neither acceptable": {
			WebhookTestData: "webhook-issue-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId1",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_deleted"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				{
					ID:        "8hduqxgiwiyi5fw3q4d6q56uho",
					ChannelID: "sampleChannelId2",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_deleted"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"status field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10004"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10004"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"status field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10005"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"status field filter configured to include all values, all are present": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10033", "10034"), Inclusion: FilterIncludeAll},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10033", "10034"), Inclusion: FilterIncludeAll},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"field filter configured to include all values, one is missing": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10033", "10035"), Inclusion: FilterIncludeAll},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"field filter configured to exclude, field is present": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10004"), Inclusion: FilterExcludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"status field filter configured to exclude, field is not present": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10005"), Inclusion: FilterExcludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet("10005"), Inclusion: FilterExcludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"filter configured to empty, field is not present": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10060", Values: NewStringSet(), Inclusion: FilterEmpty},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10060", Values: NewStringSet(), Inclusion: FilterEmpty},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"filter configured to empty, field is present": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "status", Values: NewStringSet(), Inclusion: FilterEmpty},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"custom multi-select field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10033"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10033"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"custom multi-select field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10068", Values: NewStringSet("10001"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"custom single-select field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10076", Values: NewStringSet("10039"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10076", Values: NewStringSet("10039"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"custom single-select field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10076", Values: NewStringSet("10001"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"custom string field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10078", Values: NewStringSet("some value"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10078", Values: NewStringSet("some value"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"custom string field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10078", Values: NewStringSet("wrong value"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"custom string array field filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10071", Values: NewStringSet("value1"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10071", Values: NewStringSet("value1"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"two filters, custom string array field filter with multiple values configured, one matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10071", Values: NewStringSet("value1", "value3"), Inclusion: FilterIncludeAny},
							{Key: "customfield_10071", Values: NewStringSet("value4"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"custom string array field filter with multiple values configured, one matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10071", Values: NewStringSet("value1", "value3"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10071", Values: NewStringSet("value1", "value3"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"custom string array field filter configured, does not match": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10071", Values: NewStringSet("wrong value"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"fixVersions filter configured, matches": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "fixVersions", Values: NewStringSet("10000"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "fixVersions", Values: NewStringSet("10000"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"priority filter configured, matches": {
			WebhookTestData: "webhook-server-issue-updated-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("HEY"),
						IssueTypes: NewStringSet("10001"),
						Fields: []FieldFilter{
							{Key: "Priority", Values: NewStringSet("1"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("HEY"),
						IssueTypes: NewStringSet("10001"),
						Fields: []FieldFilter{
							{Key: "Priority", Values: NewStringSet("1"), Inclusion: FilterIncludeAny},
						},
					},
					InstanceID: "jiraurl1",
				},
			},
		},
		"custom string field filter configured, field is not present in issue metadata": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "labels2", Values: NewStringSet("some value"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"custom string field filter configured, field is null in issue metadata": {
			WebhookTestData: "webhook-cloud-issue-created-many-fields.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields: []FieldFilter{
							{Key: "customfield_10026", Values: NewStringSet("some value"), Inclusion: FilterIncludeAny},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"subscribed any issue update, comment added, matches": {
			WebhookTestData: "webhook-cloud-comment-created.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
						Fields:     []FieldFilter{},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{{ChannelID: "sampleChannelId"}},
		},
		"subscribed any issue update, comment updated, matches": {
			WebhookTestData: "webhook-cloud-comment-updated.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
						Fields:     []FieldFilter{},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{{ChannelID: "sampleChannelId"}},
		},
		"subscribed any issue update, comment deleted, matches": {
			WebhookTestData: "webhook-cloud-comment-deleted.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_updated_any"),
						Projects:   NewStringSet("KT"),
						IssueTypes: NewStringSet("10002"),
						Fields:     []FieldFilter{},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{{ChannelID: "sampleChannelId"}},
		},
		"no security level provided in subscription, but security level is present in issue": {
			WebhookTestData: "webhook-issue-created-with-security-level.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
						Fields:     []FieldFilter{},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"security config disabled, no security level provided in subscription, but security level is present in issue": {
			WebhookTestData: "webhook-issue-created-with-security-level.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
						Fields:     []FieldFilter{},
					},
				},
			}),
			ChannelSubscriptions:  []ChannelSubscription{{ChannelID: "sampleChannelId"}},
			disableSecurityConfig: true,
		},
		"security level provided in subscription, but different security level is present in issue": {
			WebhookTestData: "webhook-issue-created-with-security-level.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
						Fields: []FieldFilter{
							{
								Key:       "security",
								Inclusion: FilterIncludeAll,
								Values:    NewStringSet("10002"),
							},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{},
		},
		"security level provided in subscription, and same security level is present in issue": {
			WebhookTestData: "webhook-issue-created-with-security-level.json",
			Subs: withExistingChannelSubscriptions([]ChannelSubscription{
				{
					ID:        "rg86cd65efdjdjezgisgxaitzh",
					ChannelID: "sampleChannelId",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("event_created"),
						Projects:   NewStringSet("TES"),
						IssueTypes: NewStringSet("10001"),
						Fields: []FieldFilter{
							{
								Key:       "security",
								Inclusion: FilterIncludeAll,
								Values:    NewStringSet("10001"),
							},
						},
					},
				},
			}),
			ChannelSubscriptions: []ChannelSubscription{{ChannelID: "sampleChannelId"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			p.updateConfig(func(conf *config) {
				conf.Secret = someSecret
				conf.SecurityLevelEmptyForJiraSubscriptions = !tc.disableSecurityConfig
			})
			p.SetAPI(api)

			subscriptionBytes, err := json.Marshal(tc.Subs)
			assert.Nil(t, err)

			api.On("KVGet", testSubKey).Return(subscriptionBytes, nil)

			api.On("KVCompareAndSet", testSubKey, subscriptionBytes, mock.MatchedBy(func(data []byte) bool {
				return true
			})).Return(nil)

			p.client = pluginapi.NewClient(api, p.Driver)

			data, err := getJiraTestData(tc.WebhookTestData)
			assert.Nil(t, err)

			r := bytes.NewReader(data)
			bb, err := io.ReadAll(r)
			require.Nil(t, err)

			wh, err := ParseWebhook(bb)
			assert.Nil(t, err)

			actual, err := p.getChannelsSubscribed(wh.(*webhook), testInstance1.InstanceID)
			assert.Nil(t, err)
			assert.Equal(t, len(tc.ChannelSubscriptions), len(actual))
			actualChannelIDs := NewStringSet()
			for _, channelID := range actual {
				actualChannelIDs.Add(channelID.ChannelID)
			}

			channelIDs := NewStringSet()
			for _, channelID := range tc.ChannelSubscriptions {
				channelIDs.Add(channelID.ChannelID)
			}
			assert.EqualValues(t, actualChannelIDs, channelIDs)
		})
	}
}
