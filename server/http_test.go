// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

const TestDataLongSubscriptionName = `aaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccdddddddddd`

var testSubKey = keyWithInstanceID(mockInstance1URL, JiraSubscriptionsKey)

func checkSubscriptionsEqual(t *testing.T, ls1 []ChannelSubscription, ls2 []ChannelSubscription) {
	assert.Equal(t, len(ls1), len(ls2))

	for _, a := range ls1 {
		match := false

		for _, b := range ls2 {
			if a.ID == b.ID {
				match = true
				assert.Equal(t, a.ChannelID, b.ChannelID)
				assert.True(t, a.Filters.Projects.Equals(b.Filters.Projects))
				assert.True(t, a.Filters.IssueTypes.Equals(b.Filters.IssueTypes))
				assert.True(t, a.Filters.Events.Equals(b.Filters.Events))
			}
		}
		if !match {
			assert.Fail(t, "Subscription arrays are not equal")
		}
	}
}

func checkNotSubscriptions(subsToCheck []ChannelSubscription, existing *Subscriptions, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		var existingBytes []byte
		if existing != nil {
			var err error
			existingBytes, err = json.Marshal(existing)
			assert.Nil(t, err)
		}

		api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)
		api.On("KVGet", testSubKey).Return(existingBytes, nil)

		api.On("KVCompareAndSet", testSubKey, existingBytes, mock.MatchedBy(func(data []byte) bool {
			t.Log(string(data))
			var savedSubs Subscriptions
			err := json.Unmarshal(data, &savedSubs)
			assert.Nil(t, err)

			for _, subToCheck := range subsToCheck {
				for _, savedSub := range savedSubs.Channel.ByID {
					if subToCheck.ID == savedSub.ID {
						return false
					}
				}
			}

			return true
		})).Return(true, nil)
	}
}

func checkHasSubscriptions(subsToCheck []ChannelSubscription, existing *Subscriptions, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		var existingBytes []byte
		if existing != nil {
			var err error
			existingBytes, err = json.Marshal(existing)
			assert.Nil(t, err)
		}

		api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)

		api.On("KVGet", testSubKey).Return(existingBytes, nil)

		api.On("KVCompareAndSet", testSubKey, existingBytes, mock.MatchedBy(func(data []byte) bool {
			t.Log(string(data))
			var savedSubs Subscriptions
			err := json.Unmarshal(data, &savedSubs)
			assert.Nil(t, err)

			for _, subToCheck := range subsToCheck {
				var foundSub *ChannelSubscription
				for _, savedSub := range savedSubs.Channel.ByID {
					if subToCheck.ChannelID == savedSub.ChannelID &&
						subToCheck.Filters.Projects.Equals(savedSub.Filters.Projects) &&
						subToCheck.Filters.IssueTypes.Equals(savedSub.Filters.IssueTypes) &&
						subToCheck.Filters.Events.Equals(savedSub.Filters.Events) {
						foundSub = &savedSub
						break
					}
				}

				// Check subscription exists
				if foundSub == nil {
					return false
				}

				// Check it's properly attached
				assert.Contains(t, savedSubs.Channel.IDByChannelID[foundSub.ChannelID], foundSub.ID)
				for _, event := range foundSub.Filters.Events.Elems() {
					assert.Contains(t, savedSubs.Channel.IDByEvent[event], foundSub.ID)
				}
			}

			return true
		})).Return(true, nil)
	}
}

func hasSubscriptions(subscriptions []ChannelSubscription, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		subs := withExistingChannelSubscriptions(subscriptions)

		existingBytes, err := json.Marshal(&subs)
		assert.Nil(t, err)

		api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)

		api.On("KVGet", testSubKey).Return(existingBytes, nil)
	}
}

func TestSubscribe(t *testing.T) {
	for name, tc := range map[string]struct {
		subscription       string
		expectedStatusCode int
		skipAuthorize      bool
		apiCalls           func(*plugintest.API)
	}{
		"Invalid": {
			subscription:       "{}",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			subscription:       "{}",
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Won't Decode": {
			subscription:       "{woopsie",
			expectedStatusCode: http.StatusBadRequest,
		},
		"No channel id": {
			subscription:       `{"channel_id": "badchannelid", "fields": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"Reject Ids": {
			subscription:       `{"id": "iamtryingtodosendid", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaaa", "filters": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"No Permissions": {
			subscription:       `{"channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "project": ["myproject"]}}`,
			expectedStatusCode: http.StatusForbidden,
			apiCalls: func(api *plugintest.API) {
				api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(false)
			},
		},
		"Initial Subscription happy": {
			subscription:       `{"instance_id": "jiraurl1", "name": "some name", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				{
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			}, nil, t),
		},
		"Initial Subscription, GetProject mocked error": {
			subscription:       fmt.Sprintf(`{"instance_id": "jiraurl1", "name": "some name", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["%s"], "issue_types": ["10001"]}}`, nonExistantProjectKey),
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls:           hasSubscriptions([]ChannelSubscription{}, t),
		},
		"Initial Subscription, empty name provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls:           hasSubscriptions([]ChannelSubscription{}, t),
		},
		"Initial Subscription, long name provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "` + TestDataLongSubscriptionName + `", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls:           hasSubscriptions([]ChannelSubscription{}, t),
		},
		"Initial Subscription, no project provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "somename", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": [], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls:           hasSubscriptions([]ChannelSubscription{}, t),
		},
		"Initial Subscription, no events provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "somename", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": [], "projects": ["myproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls:           hasSubscriptions([]ChannelSubscription{}, t),
		},
		"Initial Subscription, no issue types provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "somename", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": []}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls:           hasSubscriptions([]ChannelSubscription{}, t),
		},
		"Adding to existing with other channel": {
			subscription:       `{"instance_id": "jiraurl1", "name": "some name", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				{
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				{
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        model.NewId(),
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Adding to existing in same channel": {
			subscription:       `{"instance_id": "jiraurl1", "name": "subscription name", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				{
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				{
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_updated"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        model.NewId(),
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_updated"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Adding to existing with same name in same channel": {
			subscription:       `{"instance_id": "jiraurl1", "name": "SubscriptionName", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				{
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							Name:      "SubscriptionName",
							ID:        model.NewId(),
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_updated"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := Plugin{}

			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 10)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))
			api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = someSecret
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("POST", "/api/v2/subscriptions/channel", ioutil.NopCloser(bytes.NewBufferString(tc.subscription)))
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(&plugin.Context{}, w, request)
			body, _ := ioutil.ReadAll(w.Result().Body)
			t.Log(string(body))
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestDeleteSubscription(t *testing.T) {
	for name, tc := range map[string]struct {
		subscriptionID     string
		expectedStatusCode int
		skipAuthorize      bool
		apiCalls           func(*plugintest.API)
	}{
		"Invalid": {
			subscriptionID:     "blab",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			subscriptionID:     model.NewId(),
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"No Permissions": {
			subscriptionID:     "aaaaaaaaaaaaaaaaaaaaaaaaab",
			expectedStatusCode: http.StatusForbidden,
			apiCalls: func(api *plugintest.API) {
				var existingBytes []byte
				var err error
				existingBytes, err = json.Marshal(withExistingChannelSubscriptions([]ChannelSubscription{
					{
						ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("myproject"),
							IssueTypes: NewStringSet("10001"),
						},
					},
				}))
				assert.Nil(t, err)

				api.On("KVGet", testSubKey).Return(existingBytes, nil)
				api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(false)
			},
		},
		"Successful delete": {
			subscriptionID:     "aaaaaaaaaaaaaaaaaaaaaaaaab",
			expectedStatusCode: http.StatusOK,
			apiCalls: checkNotSubscriptions([]ChannelSubscription{
				{
					ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := Plugin{}

			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 10)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))
			api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = someSecret
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("DELETE",
				"/api/v2/subscriptions/channel/"+tc.subscriptionID+"?instance_id="+testInstance1.GetID().String(),
				nil)
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(&plugin.Context{}, w, request)
			body, _ := ioutil.ReadAll(w.Result().Body)
			t.Log(string(body))
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestEditSubscription(t *testing.T) {
	for name, tc := range map[string]struct {
		subscription       string
		expectedStatusCode int
		skipAuthorize      bool
		apiCalls           func(*plugintest.API)
	}{
		"Invalid": {
			subscription:       "{}",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			subscription:       "{}",
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Won't Decode": {
			subscription:       "{woopsie",
			expectedStatusCode: http.StatusBadRequest,
		},
		"No channel id": {
			subscription:       `{"id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "badchannelid", "fields": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"No ID": {
			subscription:       `{"id": "badid", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "fields": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"No Permissions": {
			subscription:       `{"id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "project": ["otherproject"]}}`,
			expectedStatusCode: http.StatusForbidden,
			apiCalls: func(api *plugintest.API) {
				api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(false)
			},
		},
		"Editing subscription": {
			subscription:       `{"instance_id": "jiraurl1", "name": "some name", "id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "projects": ["otherproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				{
					ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("otherproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, no name provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "", "id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "projects": ["otherproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, name too long": {
			subscription:       `{"instance_id": "jiraurl1", "name": "` + TestDataLongSubscriptionName + `", "id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "projects": ["otherproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, no project provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "somename", "id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "projects": [], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, no events provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "somename", "id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": [], "projects": ["otherproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, no issue types provided": {
			subscription:       `{"instance_id": "jiraurl1", "name": "somename", "id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "projects": ["otherproject"], "issue_types": []}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, GetProject mocked error. Existing sub has nonexistent project.": {
			subscription:       fmt.Sprintf(`{"instance_id": "jiraurl1", "id": "subaaaaaaaaaabbbbbbbbbbccc", "name": "subscription name", "channel_id": "channelaaaaaaaaaabbbbbbbbb", "filters": {"events": ["jira:issue_created"], "projects": ["%s"], "issue_types": ["10001"]}}`, nonExistantProjectKey),
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "subaaaaaaaaaabbbbbbbbbbccc",
							ChannelID: "channelaaaaaaaaaabbbbbbbbb",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_updated"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, GetProject mocked error. Existing sub has existing project.": {
			subscription:       fmt.Sprintf(`{"instance_id": "jiraurl1", "id": "subaaaaaaaaaabbbbbbbbbbccc", "name": "subscription name", "channel_id": "channelaaaaaaaaaabbbbbbbbb", "filters": {"events": ["jira:issue_created"], "projects": ["%s"], "issue_types": ["10001"]}}`, nonExistantProjectKey),
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						{
							ID:        "subaaaaaaaaaabbbbbbbbbbccc",
							ChannelID: "channelaaaaaaaaaabbbbbbbbb",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_updated"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := Plugin{}

			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 10)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))
			api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = someSecret
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("PUT", "/api/v2/subscriptions/channel", ioutil.NopCloser(bytes.NewBufferString(tc.subscription)))
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(&plugin.Context{}, w, request)
			body, _ := ioutil.ReadAll(w.Result().Body)
			t.Log(string(body))
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestGetSubscriptionsForChannel(t *testing.T) {
	for name, tc := range map[string]struct {
		channelID             string
		expectedStatusCode    int
		skipAuthorize         bool
		apiCalls              func(*plugintest.API)
		returnedSubscriptions []ChannelSubscription
	}{
		"Invalid": {
			channelID:          "nope",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			channelID:          model.NewId(),
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Only Subscription": {
			channelID:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []ChannelSubscription{
				{
					ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					{
						ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("myproject"),
							IssueTypes: NewStringSet("10001"),
						},
					},
				}, t),
		},
		"Multiple subscriptions": {
			channelID:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []ChannelSubscription{
				{
					ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				{
					ID:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("things"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					{
						ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("myproject"),
							IssueTypes: NewStringSet("10001"),
						},
					},
					{
						ID:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("things"),
							IssueTypes: NewStringSet("10001"),
						},
					},
				}, t),
		},
		"Only in channel": {
			channelID:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []ChannelSubscription{
				{
					ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					{
						ID:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("myproject"),
							IssueTypes: NewStringSet("10001"),
						},
					},
					{
						ID:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelID: "aaaaaaaaaaaaaaaaaaaaaaaaad",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("things"),
							IssueTypes: NewStringSet("10001"),
						},
					},
				}, t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := Plugin{}

			api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 10)...).Return(nil)
			api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = someSecret
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v2/subscriptions/channel/"+tc.channelID+"?instance_id="+testInstance1.GetID().String(), nil)
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-ID", model.NewId())
			}
			p.ServeHTTP(&plugin.Context{}, w, request)
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)

			if tc.returnedSubscriptions != nil {
				subscriptions := []ChannelSubscription{}
				body, _ := ioutil.ReadAll(w.Result().Body)
				err := json.NewDecoder(bytes.NewReader(body)).Decode(&subscriptions)
				assert.Nil(t, err)
				checkSubscriptionsEqual(t, tc.returnedSubscriptions, subscriptions)
			}
		})
	}
}
