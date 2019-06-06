// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const (
	JIRA_WEBHOOK_EVENT_ISSUE_CREATED = "jira:issue_created"
	JIRA_WEBHOOK_EVENT_ISSUE_UPDATED = "jira:issue_updated"
	JIRA_WEBHOOK_EVENT_ISSUE_DELETED = "jira:issue_deleted"
)

const JiraSubscriptionsKey = "jirasub"

type ChannelSubscription struct {
	Id        string              `json:"id"`
	ChannelId string              `json:"channel_id"`
	Filters   map[string][]string `json:"filters"`
}

type ChannelSubscriptions struct {
	ById          map[string]ChannelSubscription `json:"by_id"`
	IdByChannelId map[string][]string            `json:"id_by_channel_id"`
	IdByEvent     map[string][]string            `json:"id_by_event"`
}

func NewChannelSubscriptions() *ChannelSubscriptions {
	return &ChannelSubscriptions{
		ById:          map[string]ChannelSubscription{},
		IdByChannelId: map[string][]string{},
		IdByEvent:     map[string][]string{},
	}
}

func (s *ChannelSubscriptions) remove(sub *ChannelSubscription) {
	delete(s.ById, sub.Id)

	remove := func(ids []string, idToRemove string) []string {
		for i, id := range ids {
			if id == idToRemove {
				ids[i] = ids[len(ids)-1]
				return ids[:len(ids)-1]
			}
		}
		return ids
	}

	s.IdByChannelId[sub.ChannelId] = remove(s.IdByChannelId[sub.ChannelId], sub.Id)

	for _, event := range sub.Filters["events"] {
		s.IdByEvent[event] = remove(s.IdByEvent[event], sub.Id)
	}
}

func (s *ChannelSubscriptions) add(newSubscription *ChannelSubscription) {
	s.ById[newSubscription.Id] = *newSubscription
	s.IdByChannelId[newSubscription.ChannelId] = append(s.IdByChannelId[newSubscription.ChannelId], newSubscription.Id)
	for _, event := range newSubscription.Filters["events"] {
		s.IdByEvent[event] = append(s.IdByEvent[event], newSubscription.Id)
	}
}

type Subscriptions struct {
	Channel *ChannelSubscriptions
}

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		Channel: NewChannelSubscriptions(),
	}
}

func SubscriptionsFromJson(bytes []byte) (*Subscriptions, error) {
	var subs *Subscriptions
	if len(bytes) != 0 {
		unmarshalErr := json.Unmarshal(bytes, &subs)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}
	} else {
		subs = NewSubscriptions()
	}

	return subs, nil
}

func GetBotUserID(conf Config, api plugin.API) (string, error) {
	user, appErr := api.GetUserByUsername(conf.UserName)
	if appErr != nil {
		return "", fmt.Errorf(appErr.Message)
	}

	return user.Id, nil
}

func getChannelsSubscribed(api plugin.API, jwh *JiraWebhook) ([]string, error) {
	subs, err := getSubscriptions(api)
	if err != nil {
		return nil, err
	}

	subIds := subs.Channel.IdByEvent[jwh.WebhookEvent]

	channelIds := []string{}
	for _, subId := range subIds {
		sub := subs.Channel.ById[subId]

		acceptable := true
		for field, acceptableValues := range sub.Filters {
			// Blank in acceptable values means all values are acceptable
			if len(acceptableValues) == 0 {
				continue
			}
			switch field {
			case "event":
				found := false
				for _, acceptableEvent := range acceptableValues {
					if acceptableEvent == jwh.WebhookEvent {
						found = true
						break
					}
				}
				if !found {
					acceptable = false
					break
				}
			case "project":
				found := false
				for _, acceptableProject := range acceptableValues {
					if acceptableProject == jwh.Issue.Fields.Project.Key {
						found = true
						break
					}
				}
				if !found {
					acceptable = false
					break
				}
			case "issue_type":
				found := false
				for _, acceptableIssueType := range acceptableValues {
					if acceptableIssueType == jwh.Issue.Fields.Type.ID {
						found = true
						break
					}
				}
				if !found {
					acceptable = false
					break
				}
			}
		}

		if acceptable {
			channelIds = append(channelIds, sub.ChannelId)
		}
	}

	return channelIds, nil
}

func getSubscriptions(api plugin.API) (*Subscriptions, error) {
	data, err := api.KVGet(JiraSubscriptionsKey)
	if err != nil {
		return nil, err
	}
	return SubscriptionsFromJson(data)
}

func getSubscriptionsForChannel(api plugin.API, channelId string) ([]ChannelSubscription, error) {
	subs, err := getSubscriptions(api)
	if err != nil {
		return nil, err
	}

	channelSubscriptions := []ChannelSubscription{}
	for _, channelSubscriptionId := range subs.Channel.IdByChannelId[channelId] {
		channelSubscriptions = append(channelSubscriptions, subs.Channel.ById[channelSubscriptionId])
	}

	return channelSubscriptions, nil
}

func getChannelSubscription(api plugin.API, subscriptionId string) (*ChannelSubscription, error) {
	subs, err := getSubscriptions(api)
	if err != nil {
		return nil, err
	}

	subscription, ok := subs.Channel.ById[subscriptionId]
	if !ok {
		return nil, errors.New("could not find subscription")
	}

	return &subscription, nil
}

func removeChannelSubscription(api plugin.API, subscriptionId string) error {
	return atomicModify(api, JiraSubscriptionsKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		subscription, ok := subs.Channel.ById[subscriptionId]
		if !ok {
			return nil, errors.New("could not find subscription")
		}

		subs.Channel.remove(&subscription)

		modifiedBytes, marshalErr := json.Marshal(&subs)
		if marshalErr != nil {
			return nil, marshalErr
		}

		return modifiedBytes, nil
	})
}

func addChannelSubscription(api plugin.API, newSubscription *ChannelSubscription) error {
	return atomicModify(api, JiraSubscriptionsKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		newSubscription.Id = model.NewId()
		subs.Channel.add(newSubscription)

		modifiedBytes, marshalErr := json.Marshal(&subs)
		if marshalErr != nil {
			return nil, marshalErr
		}

		return modifiedBytes, nil
	})
}

func editChannelSubscription(api plugin.API, modifiedSubscription *ChannelSubscription) error {
	return atomicModify(api, JiraSubscriptionsKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		oldSub, ok := subs.Channel.ById[modifiedSubscription.Id]
		if !ok {
			return nil, errors.New("Existing subscription does not exist.")
		}
		subs.Channel.remove(&oldSub)
		subs.Channel.add(modifiedSubscription)

		modifiedBytes, marshalErr := json.Marshal(&subs)
		if marshalErr != nil {
			return nil, marshalErr
		}

		return modifiedBytes, nil
	})
}

func atomicModify(api plugin.API, key string, modify func(initialValue []byte) ([]byte, error)) error {
	readModify := func() ([]byte, []byte, error) {
		initialBytes, appErr := api.KVGet(key)
		if appErr != nil {
			return nil, nil, errors.Wrap(appErr, "unable to read inital value")
		}

		modifiedBytes, err := modify(initialBytes)
		if err != nil {
			return nil, nil, errors.Wrap(err, "modification error")
		}

		return initialBytes, modifiedBytes, nil
	}

	success := false
	for !success {
		//initialBytes, newValue, err := readModify()
		_, newValue, err := readModify()
		if err != nil {
			return err
		}

		var setError *model.AppError
		// Commenting this out so we can support < 5.12 for 2.0
		//success, setError = p.API.KVCompareAndSet(key, initialBytes, newValue)
		setError = api.KVSet(key, newValue)
		success = true
		if setError != nil {
			return errors.Wrap(setError, "problem writing value")
		}

	}

	return nil
}

var httpSubscribeWebhook = []ActionFunc{
	RequireHTTPPost,
	RequireInstance,
	handleSubscribeWebhook,
}

func handleSubscribeWebhook(a *Action) error {
	if a.PluginConfig.Secret == "" || a.PluginConfig.UserName == "" {
		return a.RespondError(http.StatusForbidden, nil,
			"Jira plugin not configured correctly; must provide Secret and UserName")
	}

	if subtle.ConstantTimeCompare(
		[]byte(a.HTTPRequest.URL.Query().Get("secret")),
		[]byte(a.PluginConfig.Secret)) != 1 {
		return a.RespondError(http.StatusForbidden, nil,
			"request URL: secret did not match")
	}

	wh, jwh, err := ParseWebhook(a.HTTPRequest.Body)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	botUserId, err := GetBotUserID(a.PluginConfig, a.API)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	channelIds, err := getChannelsSubscribed(a.API, jwh)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	for _, channelId := range channelIds {
		if _, status, err1 := wh.PostToChannel(a.API, channelId, botUserId); err1 != nil {
			return a.RespondError(status, err1)
		}
	}

	_, status, err := wh.PostNotifications(a.PluginConfig, a.API, a.UserStore, a.Instance)
	if err != nil {
		return a.RespondError(status, err)
	}

	return nil
}

var httpChannelSubscriptions = []ActionFunc{
	RequireMattermostUserId,
	handleChannelSubscriptions,
}

func handleChannelSubscriptions(a *Action) error {
	switch a.HTTPRequest.Method {
	case http.MethodPost:
		return handleChannelCreateSubscription(a)
	case http.MethodDelete:
		return handleChannelDeleteSubscription(a)
	case http.MethodGet:
		return handleChannelGetSubscriptions(a)
	case http.MethodPut:
		return handleChannelEditSubscription(a)
	default:
		return a.RespondError(http.StatusMethodNotAllowed, nil, "Request: %q is not allowed.", a.HTTPRequest.Method)
	}
}

func handleChannelCreateSubscription(a *Action) error {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(a.HTTPRequest.Body).Decode(&subscription)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err,
			"failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 0 {
		return a.RespondError(http.StatusBadRequest, nil,
			"Channel subscription invalid")
	}

	_, appErr := a.API.GetChannelMember(subscription.ChannelId, a.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	if err := addChannelSubscription(a.API, &subscription); err != nil {
		a.RespondError(http.StatusInternalServerError, err)
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func handleChannelEditSubscription(a *Action) error {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(a.HTTPRequest.Body).Decode(&subscription)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err,
			"failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"Channel subscription invalid")
	}

	if _, appErr := a.API.GetChannelMember(subscription.ChannelId, a.MattermostUserId); appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	if err := editChannelSubscription(a.API, &subscription); err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func handleChannelDeleteSubscription(a *Action) error {
	// routeAPISubscriptionsChannel has the trailing '/'
	subscriptionId := strings.TrimPrefix(a.HTTPRequest.URL.Path, routeAPISubscriptionsChannel)
	if len(subscriptionId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad subscription id")
	}

	subscription, err := getChannelSubscription(a.API, subscriptionId)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err, "bad subscription id")
	}

	_, appErr := a.API.GetChannelMember(subscription.ChannelId, a.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	if err := removeChannelSubscription(a.API, subscriptionId); err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"unable to remove channel subscription")
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func handleChannelGetSubscriptions(a *Action) error {
	// routeAPISubscriptionsChannel has the trailing '/'
	channelId := strings.TrimPrefix(a.HTTPRequest.URL.Path, routeAPISubscriptionsChannel)
	if len(channelId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad channel id")
	}

	if _, appErr := a.API.GetChannelMember(channelId, a.MattermostUserId); appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	subscriptions, err := getSubscriptionsForChannel(a.API, channelId)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"unable to get channel subscriptions")
	}

	return a.RespondJSON(subscriptions)
}
