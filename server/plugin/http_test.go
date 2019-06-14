// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

func checkNotSubscriptions(subsToCheck []ChannelSubscription, existing *Subscriptions, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		var existingBytes []byte
		if existing != nil {
			var err error
			existingBytes, err = json.Marshal(existing)
			assert.Nil(t, err)
		}

		api.On("KVGet", JiraSubscriptionsKey).Return(existingBytes, nil)

		// Temp changes to revert when we can use KVCompareAndSet
		//api.On("KVCompareAndSet", JiraSubscriptionsKey, existingBytes, mock.MatchedBy(func(data []byte) bool {
		api.On("KVSet", JiraSubscriptionsKey, mock.MatchedBy(func(data []byte) bool {
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
			//})).Return(true, nil)
		})).Return(nil)
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

		api.On("KVGet", JiraSubscriptionsKey).Return(existingBytes, nil)

		// Temp changes to revert when we can use KVCompareAndSet
		//api.On("KVCompareAndSet", JiraSubscriptionsKey, existingBytes, mock.MatchedBy(func(data []byte) bool {
		api.On("KVSet", JiraSubscriptionsKey, mock.MatchedBy(func(data []byte) bool {
			t.Log(string(data))
			var savedSubs Subscriptions
			err := json.Unmarshal(data, &savedSubs)
			assert.Nil(t, err)

			for _, subToCheck := range subsToCheck {
				var foundSub *ChannelSubscription
				for _, savedSub := range savedSubs.Channel.ById {
					if subToCheck.ChannelId == savedSub.ChannelId && reflect.DeepEqual(subToCheck.Filters, savedSub.Filters) {
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
				for _, event := range foundSub.Filters["events"] {
					assert.Contains(t, savedSubs.Channel.IdByEvent[event], foundSub.Id)
				}
			}

			return true
			//})).Return(true, nil)
		})).Return(nil)
	}
}

func withExistingChannelSubscriptions(subscriptions []ChannelSubscription) *Subscriptions {
	ret := NewSubscriptions()
	for _, sub := range subscriptions {
		ret.Channel.add(&sub)
	}
	return ret
}

func hasSubscriptions(subscriptions []ChannelSubscription, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		subs := withExistingChannelSubscriptions(subscriptions)

		existingBytes, err := json.Marshal(&subs)
		assert.Nil(t, err)

		api.On("KVGet", JiraSubscriptionsKey).Return(existingBytes, nil)
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
		"Initial Subscription": {
			subscription:       `{"channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "project": ["myproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			}, nil, t),
		},
		"Adding to existing with other channel": {
			subscription:       `{"channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "project": ["myproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        model.NewId(),
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
							},
						},
					}), t),
		},
		"Adding to existing in same channel": {
			subscription:       `{"channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "project": ["myproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
				ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_updated"},
						"project": []string{"myproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        model.NewId(),
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_updated"},
								"project": []string{"myproject"},
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

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.UpdateConfig(func(conf *Config) {
				conf.Secret = "somesecret"
				conf.UserName = "someuser"
			})
			p.SetAPI(api)
			p.CurrentInstanceStore = mockCurrentInstanceStore{&p}

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
		"Sucessfull delete": {
			subscriptionId:     "aaaaaaaaaaaaaaaaaaaaaaaaab",
			expectedStatusCode: http.StatusOK,
			apiCalls: checkNotSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
							},
						},
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
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

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.UpdateConfig(func(conf *Config) {
				conf.Secret = "somesecret"
				conf.UserName = "someuser"
			})
			p.SetAPI(api)
			p.CurrentInstanceStore = mockCurrentInstanceStore{&p}

			w := httptest.NewRecorder()
			request := httptest.NewRequest("DELETE", "/api/v2/subscriptions/channel/"+tc.subscriptionId, nil)
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
		"Editing subscription": {
			subscription:       `{"id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "project": ["otherproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]ChannelSubscription{
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"otherproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]ChannelSubscription{
						ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
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

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			p.UpdateConfig(func(conf *Config) {
				conf.Secret = "somesecret"
				conf.UserName = "someuser"
			})
			p.SetAPI(api)
			p.CurrentInstanceStore = mockCurrentInstanceStore{&p}

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
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"myproject"},
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
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
				ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"things"},
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"myproject"},
						},
					},
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"things"},
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
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]ChannelSubscription{
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"myproject"},
						},
					},
					ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaad",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"things"},
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

			p.UpdateConfig(func(conf *Config) {
				conf.Secret = "somesecret"
				conf.UserName = "someuser"
			})
			p.SetAPI(api)
			p.CurrentInstanceStore = mockCurrentInstanceStore{&p}

			w := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v2/subscriptions/channel/"+tc.channelId, nil)
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

				assert.Equal(t, tc.returnedSubscriptions, subscriptions)
			}
		})
	}
}
