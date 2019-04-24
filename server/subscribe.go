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
	var cs ChannelSubscriptions
	cs.ById = make(map[string]ChannelSubscription)
	cs.IdByChannelId = make(map[string][]string)
	cs.IdByEvent = make(map[string][]string)
	return &cs
}

func (s *ChannelSubscriptions) remove(sub *ChannelSubscription) {
	delete(s.ById, sub.Id)

	iToRemove := -1
	for i, subId := range s.IdByChannelId[sub.ChannelId] {
		if subId == sub.Id {
			iToRemove = i
			break
		}
	}
	s.IdByChannelId[sub.ChannelId][iToRemove] = s.IdByChannelId[sub.ChannelId][len(s.IdByChannelId[sub.ChannelId])-1]
	s.IdByChannelId[sub.ChannelId] = s.IdByChannelId[sub.ChannelId][:len(s.IdByChannelId[sub.ChannelId])-1]

	for _, event := range sub.Filters["events"] {
		iToRemove := -1
		for i, subId := range s.IdByEvent[event] {
			if subId == sub.Id {
				iToRemove = i
				break
			}
		}
		s.IdByEvent[event][iToRemove] = s.IdByEvent[event][len(s.IdByEvent[event])-1]
		s.IdByEvent[event] = s.IdByEvent[event][:len(s.IdByEvent[event])-1]
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
	Channel ChannelSubscriptions
}

func NewSubscriptions() *Subscriptions {
	var subs Subscriptions
	subs.Channel = *NewChannelSubscriptions()
	return &subs
}

func SubscriptionsFromJson(bytes []byte) (*Subscriptions, error) {
	var subs Subscriptions
	if len(bytes) != 0 {
		unmarshalErr := json.Unmarshal(bytes, &subs)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}
	} else {
		subs = *NewSubscriptions()
	}

	return &subs, nil
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
				for _, acceptableEvent := range acceptableValues {
					if acceptableEvent == webhook.Issue.Fields.Project.Key {
						found = true
						break
					}
				}
				if !found {
					acceptable = false
					break
				}
			default:
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
		initialBytes, newValue, err := readModify()
		if err != nil {
			return err
		}

		var setError *model.AppError
		success, setError = p.API.KVCompareAndSet(key, initialBytes, newValue)
		if setError != nil {
			return errors.Wrap(setError, "problem writing value")
		}

	}

	return nil
}

func httpSubscribeWebhook(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}

	cfg := p.getConfig()
	if cfg.Secret == "" || cfg.UserName == "" {
		return http.StatusForbidden, fmt.Errorf("JIRA plugin not configured correctly; must provide Secret and UserName")
	}

	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(cfg.Secret)) != 1 {
		return http.StatusForbidden, fmt.Errorf("Request URL: secret did not match")
	}

	parsed, err := parse(r.Body, nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	botUserId, err := p.getUserID()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	channelIds, err := p.getChannelsSubscribed(parsed)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	attachment := newSlackAttachment(parsed)

	for _, channelId := range channelIds {
		post := &model.Post{
			ChannelId: channelId,
			UserId:    botUserId,
		}

		model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})

		if err != nil {
			return http.StatusBadGateway, err
		}
		_, appErr := p.API.CreatePost(post)
		if appErr != nil {
			return appErr.StatusCode, fmt.Errorf(appErr.Message)
		}
	}

	return 200, nil
}

func httpChannelCreateSubscription(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 0 {
		return http.StatusBadRequest, fmt.Errorf("Channel subscription invalid")
	}

	if err := p.addChannelSubscription(&subscription); err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"status\": \"OK\"}"))

	return 200, nil
}

func httpChannelEditSubscription(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 26 {
		return http.StatusBadRequest, fmt.Errorf("Channel subscription invalid")
	}

	if err := p.editChannelSubscription(&subscription); err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"status\": \"OK\"}"))

	return 200, nil
}

func httpChannelDeleteSubscription(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	subscriptionId := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(subscriptionId) != 26 {
		return http.StatusBadRequest, errors.New("bad subscription id")
	}

	if err := p.removeChannelSubscription(subscriptionId); err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "unable to remove channel subscription")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"status\": \"OK\"}"))

	return 200, nil
}

func httpChannelGetSubscriptions(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	channelId := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(channelId) != 26 {
		return http.StatusBadRequest, errors.New("bad channel id")
	}

	subscriptions, err := p.getSubscriptionsForChannel(channelId)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "unable to get channel subscriptions")
	}

	bytes, err := json.Marshal(subscriptions)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "unable to marshal subscriptions")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)

	return 200, nil
}

func httpChannelSubscriptions(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	switch r.Method {
	case http.MethodPost:
		return httpChannelCreateSubscription(p, w, r)
	case http.MethodDelete:
		return httpChannelDeleteSubscription(p, w, r)
	case http.MethodGet:
		return httpChannelGetSubscriptions(p, w, r)
	case http.MethodPut:
		return httpChannelEditSubscription(p, w, r)
	default:
		return http.StatusMethodNotAllowed, fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}
}
