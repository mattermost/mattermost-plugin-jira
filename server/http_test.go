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

const TEST_DATA_LONG_SUBSCRIPTION_NAME = `aaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccddddddddddaaaaaaaaaabbbbbbbbbbccccccccccdddddddddd`

var testSubKey = keyWithInstanceID(mockInstance1URL, JIRA_SUBSCRIPTIONS_KEY)

func checkSubscriptionsEqual(t *testing.T, ls1 []ChannelSubscription, ls2 []ChannelSubscription) {
	assert.Equal(t, len(ls1), len(ls2))

	for _, a := range ls1 {
		match := false

		for _, b := range ls2 {
			if a.Id == b.Id {
				match = true
				assert.Equal(t, a.ChannelId, b.ChannelId)
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
				for _, savedSub := range savedSubs.Channel.ById {
					if subToCheck.Id == savedSub.Id {
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
				for _, savedSub := range savedSubs.Channel.ById {
					if subToCheck.ChannelId == savedSub.ChannelId &&
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
				assert.Contains(t, savedSubs.Channel.IdByChannelId[foundSub.ChannelId], foundSub.Id)
				for _, event := range foundSub.Filters.Events.Elems() {
					assert.Contains(t, savedSubs.Channel.IdByEvent[event], foundSub.Id)
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
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
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
			subscription:       `{"instance_id": "jiraurl1", "name": "` + TEST_DATA_LONG_SUBSCRIPTION_NAME + `", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "projects": ["myproject"], "issue_types": ["10001"]}}`,
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
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        model.NewId(),
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
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
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_updated"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        model.NewId(),
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
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
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Name:      "SubscriptionName",
							Id:        model.NewId(),
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
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

			api.On("LogDebug",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))
			api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = "somesecret"
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("POST", "/api/v2/subscriptions/channel", ioutil.NopCloser(bytes.NewBufferString(tc.subscription)))
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
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
		subscriptionId     string
		expectedStatusCode int
		skipAuthorize      bool
		apiCalls           func(*plugintest.API)
	}{
		"Invalid": {
			subscriptionId:     "blab",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			subscriptionId:     model.NewId(),
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"No Permissions": {
			subscriptionId:     "aaaaaaaaaaaaaaaaaaaaaaaaab",
			expectedStatusCode: http.StatusForbidden,
			apiCalls: func(api *plugintest.API) {
				var existingBytes []byte
				var err error
				existingBytes, err = json.Marshal(withExistingChannelSubscriptions([]ChannelSubscription{
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
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
		"Sucessful delete": {
			subscriptionId:     "aaaaaaaaaaaaaaaaaaaaaaaaab",
			expectedStatusCode: http.StatusOK,
			apiCalls: checkNotSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
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

			api.On("LogDebug",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))
			api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = "somesecret"
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("DELETE",
				"/api/v2/subscriptions/channel/"+tc.subscriptionId+"?instance_id="+testInstance1.GetID().String(),
				nil)
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
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
		"No Id": {
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
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("otherproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
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
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: SubscriptionFilters{
								Events:     NewStringSet("jira:issue_created"),
								Projects:   NewStringSet("myproject"),
								IssueTypes: NewStringSet("10001"),
							},
						},
					}), t),
		},
		"Editing subscription, name too long": {
			subscription:       `{"instance_id": "jiraurl1", "name": "` + TEST_DATA_LONG_SUBSCRIPTION_NAME + `", "id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "projects": ["otherproject"], "issue_types": ["10001"]}}`,
			expectedStatusCode: http.StatusInternalServerError,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
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
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
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
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
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
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
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
						ChannelSubscription{
							Id:        "subaaaaaaaaaabbbbbbbbbbccc",
							ChannelId: "channelaaaaaaaaaabbbbbbbbb",
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
						ChannelSubscription{
							Id:        "subaaaaaaaaaabbbbbbbbbbccc",
							ChannelId: "channelaaaaaaaaaabbbbbbbbb",
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

			api.On("LogDebug",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))
			api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = "somesecret"
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("PUT", "/api/v2/subscriptions/channel", ioutil.NopCloser(bytes.NewBufferString(tc.subscription)))
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
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
		channelId             string
		expectedStatusCode    int
		skipAuthorize         bool
		apiCalls              func(*plugintest.API)
		returnedSubscriptions []ChannelSubscription
	}{
		"Invalid": {
			channelId:          "nope",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			channelId:          model.NewId(),
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Only Subscription": {
			channelId:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []ChannelSubscription{
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("myproject"),
							IssueTypes: NewStringSet("10001"),
						},
					},
				}, t),
		},
		"Multiple subscriptions": {
			channelId:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []ChannelSubscription{
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("things"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("myproject"),
							IssueTypes: NewStringSet("10001"),
						},
					},
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("things"),
							IssueTypes: NewStringSet("10001"),
						},
					},
				}, t),
		},
		"Only in channel": {
			channelId:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []ChannelSubscription{
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: SubscriptionFilters{
						Events:     NewStringSet("jira:issue_created"),
						Projects:   NewStringSet("myproject"),
						IssueTypes: NewStringSet("10001"),
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: SubscriptionFilters{
							Events:     NewStringSet("jira:issue_created"),
							Projects:   NewStringSet("myproject"),
							IssueTypes: NewStringSet("10001"),
						},
					},
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaad",
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

			api.On("LogDebug",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.updateConfig(func(conf *config) {
				conf.Secret = "somesecret"
			})
			p.SetAPI(api)
			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v2/subscriptions/channel/"+tc.channelId+"?instance_id="+testInstance1.GetID().String(), nil)
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
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
