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
	"github.com/pkg/errors"
)

const (
	JIRA_WEBHOOK_EVENT_ISSUE_CREATED = "jira:issue_created"
	JIRA_WEBHOOK_EVENT_ISSUE_UPDATED = "jira:issue_updated"
	JIRA_WEBHOOK_EVENT_ISSUE_DELETED = "jira:issue_deleted"

	JIRA_SUBSCRIPTIONS_KEY = "jirasub"
)

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

func (p *Plugin) getUserID() (string, error) {
	cfg := p.getConfig()
	user, appErr := p.API.GetUserByUsername(cfg.UserName)
	if appErr != nil {
		return "", fmt.Errorf(appErr.Message)
	}

	return user.Id, nil
}

func (p *Plugin) getChannelsSubscribed(webhook *parsedJIRAWebhook) ([]string, error) {
	subs, err := p.getSubscriptions()
	if err != nil {
		return nil, err
	}

	subIds := subs.Channel.IdByEvent[webhook.WebhookEvent]

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
					if acceptableEvent == webhook.WebhookEvent {
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
					if acceptableProject == webhook.Issue.Fields.Project.Key {
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
					if acceptableIssueType == webhook.Issue.Fields.IssueType.Id {
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

func (p *Plugin) getSubscriptions() (*Subscriptions, error) {
	data, err := p.API.KVGet(JIRA_SUBSCRIPTIONS_KEY)
	if err != nil {
		return nil, err
	}
	return SubscriptionsFromJson(data)
}

func (p *Plugin) getSubscriptionsForChannel(channelId string) ([]ChannelSubscription, error) {
	subs, err := p.getSubscriptions()
	if err != nil {
		return nil, err
	}

	channelSubscriptions := []ChannelSubscription{}
	for _, channelSubscriptionId := range subs.Channel.IdByChannelId[channelId] {
		channelSubscriptions = append(channelSubscriptions, subs.Channel.ById[channelSubscriptionId])
	}

	return channelSubscriptions, nil
}

func (p *Plugin) getChannelSubscription(subscriptionId string) (*ChannelSubscription, error) {
	subs, err := p.getSubscriptions()
	if err != nil {
		return nil, err
	}

	subscription, ok := subs.Channel.ById[subscriptionId]
	if !ok {
		return nil, errors.New("could not find subscription")
	}

	return &subscription, nil
}

func (p *Plugin) removeChannelSubscription(subscriptionId string) error {
	return p.atomicModify(JIRA_SUBSCRIPTIONS_KEY, func(initialBytes []byte) ([]byte, error) {
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

func (p *Plugin) addChannelSubscription(newSubscription *ChannelSubscription) error {
	return p.atomicModify(JIRA_SUBSCRIPTIONS_KEY, func(initialBytes []byte) ([]byte, error) {
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

func (p *Plugin) editChannelSubscription(modifiedSubscription *ChannelSubscription) error {
	return p.atomicModify(JIRA_SUBSCRIPTIONS_KEY, func(initialBytes []byte) ([]byte, error) {
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

func (p *Plugin) atomicModify(key string, modify func(initialValue []byte) ([]byte, error)) error {
	readModify := func() ([]byte, []byte, error) {
		initialBytes, appErr := p.API.KVGet(key)
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
		setError = p.API.KVSet(key, newValue)
		success = true
		if setError != nil {
			return errors.Wrap(setError, "problem writing value")
		}

	}

	return nil
}

func httpSubscribeWebhook(a *Action) error {
	err := RequireHTTPPost(a)
	if err != nil {
		return err
	}

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

	parsed, err := parse(a.HTTPRequest.Body, nil)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	botUserId, err := a.Plugin.getUserID()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	channelIds, err := a.Plugin.getChannelsSubscribed(parsed)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	attachment := newSlackAttachment(parsed)

	for _, channelId := range channelIds {
		post := &model.Post{
			ChannelId: channelId,
			UserId:    botUserId,
		}

		model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})

		if err != nil {
			return a.RespondError(http.StatusBadGateway, err)
		}
		_, appErr := a.Plugin.API.CreatePost(post)
		if appErr != nil {
			return a.RespondError(appErr.StatusCode, appErr)
		}
	}
	return nil
}

func httpChannelCreateSubscription(a *Action) error {
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

	_, appErr := a.Plugin.API.GetChannelMember(subscription.ChannelId, a.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	if err := a.Plugin.addChannelSubscription(&subscription); err != nil {
		a.RespondError(http.StatusInternalServerError, err)
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func httpChannelEditSubscription(a *Action) error {
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

	if _, appErr := a.Plugin.API.GetChannelMember(subscription.ChannelId, a.MattermostUserId); appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	if err := a.Plugin.editChannelSubscription(&subscription); err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func httpChannelDeleteSubscription(a *Action) error {
	subscriptionId := strings.TrimPrefix(a.HTTPRequest.URL.Path,
		strings.TrimSuffix(routeAPISubscriptionsChannel, "*"))
	if len(subscriptionId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad subscription id")
	}

	subscription, err := a.Plugin.getChannelSubscription(subscriptionId)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err, "bad subscription id")
	}

	_, appErr := a.Plugin.API.GetChannelMember(subscription.ChannelId, a.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	if err := a.Plugin.removeChannelSubscription(subscriptionId); err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"unable to remove channel subscription")
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func httpChannelGetSubscriptions(a *Action) error {
	channelId := strings.TrimPrefix(a.HTTPRequest.URL.Path,
		strings.TrimSuffix(routeAPISubscriptionsChannel, "*"))
	if len(channelId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad channel id")
	}

	if _, appErr := a.Plugin.API.GetChannelMember(channelId, a.MattermostUserId); appErr != nil {
		return a.RespondError(http.StatusForbidden, nil,
			"Not a member of the channel specified")
	}

	subscriptions, err := a.Plugin.getSubscriptionsForChannel(channelId)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"unable to get channel subscriptions")
	}

	return a.RespondJSON(subscriptions)
}

func httpChannelSubscriptions(a *Action) error {
	switch a.HTTPRequest.Method {
	case http.MethodPost:
		return httpChannelCreateSubscription(a)
	case http.MethodDelete:
		return httpChannelDeleteSubscription(a)
	case http.MethodGet:
		return httpChannelGetSubscriptions(a)
	case http.MethodPut:
		return httpChannelEditSubscription(a)
	default:
		return a.RespondError(http.StatusMethodNotAllowed, nil, "Request: %q is not allowed.", a.HTTPRequest.Method)
	}
}
