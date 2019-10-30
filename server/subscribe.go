// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

const (
	JIRA_SUBSCRIPTIONS_KEY = "jirasub"

	FILTER_INCLUDE_ANY = "include_any"
	FILTER_INCLUDE_ALL = "include_all"
	FILTER_EXCLUDE_ANY = "exclude_any"
)

type FieldFilter struct {
	Key       string    `json:"key"`
	Inclusion string    `json:"inclusion"`
	Values    StringSet `json:"values"`
}

type SubscriptionFilters struct {
	Events     StringSet     `json:"events"`
	Projects   StringSet     `json:"projects"`
	IssueTypes StringSet     `json:"issue_types"`
	Fields     []FieldFilter `json:"fields"`
}

type ChannelSubscription struct {
	Id        string              `json:"id"`
	ChannelId string              `json:"channel_id"`
	Filters   SubscriptionFilters `json:"filters"`
	Name      string              `json:"name"`
}

type ChannelSubscriptions struct {
	ById          map[string]ChannelSubscription `json:"by_id"`
	IdByChannelId map[string]StringSet           `json:"id_by_channel_id"`
	IdByEvent     map[string]StringSet           `json:"id_by_event"`
}

func NewChannelSubscriptions() *ChannelSubscriptions {
	return &ChannelSubscriptions{
		ById:          map[string]ChannelSubscription{},
		IdByChannelId: map[string]StringSet{},
		IdByEvent:     map[string]StringSet{},
	}
}

func (s *ChannelSubscriptions) remove(sub *ChannelSubscription) {
	delete(s.ById, sub.Id)

	s.IdByChannelId[sub.ChannelId] = s.IdByChannelId[sub.ChannelId].Subtract(sub.Id)

	for _, event := range sub.Filters.Events.Elems() {
		s.IdByEvent[event] = s.IdByEvent[event].Subtract(sub.Id)
	}
}

func (s *ChannelSubscriptions) add(newSubscription *ChannelSubscription) {
	s.ById[newSubscription.Id] = *newSubscription
	s.IdByChannelId[newSubscription.ChannelId] = s.IdByChannelId[newSubscription.ChannelId].Add(newSubscription.Id)
	for _, event := range newSubscription.Filters.Events.Elems() {
		s.IdByEvent[event] = s.IdByEvent[event].Add(newSubscription.Id)
	}
}

type Subscriptions struct {
	PluginVersion string
	Channel       *ChannelSubscriptions
}

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		PluginVersion: manifest.Version,
		Channel:       NewChannelSubscriptions(),
	}
}

func SubscriptionsFromJson(bytes []byte) (*Subscriptions, error) {
	var subs *Subscriptions
	if len(bytes) != 0 {
		unmarshalErr := json.Unmarshal(bytes, &subs)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}
		subs.PluginVersion = manifest.Version
	} else {
		subs = NewSubscriptions()
	}

	return subs, nil
}

func (p *Plugin) getUserID() string {
	return p.getConfig().botUserID
}

func (p *Plugin) getChannelsSubscribed(wh *webhook) (StringSet, error) {
	jwh := wh.JiraWebhook
	subs, err := p.getSubscriptions()
	if err != nil {
		return nil, err
	}

	webhookEvents := wh.Events()
	subIds := subs.Channel.ById

	channelIds := NewStringSet()
	for _, sub := range subIds {
		foundEvent := false
		eventTypes := sub.Filters.Events
		if eventTypes.Intersection(webhookEvents).Len() > 0 {
			foundEvent = true
		} else if eventTypes.ContainsAny(eventUpdatedAny) {
			for _, eventType := range webhookEvents.Elems() {
				if strings.HasPrefix(eventType, "event_updated") {
					foundEvent = true
				}
			}
		}

		if !foundEvent {
			continue
		}

		if !sub.Filters.IssueTypes.ContainsAny(jwh.Issue.Fields.Type.ID) {
			continue
		}

		if !sub.Filters.Projects.ContainsAny(jwh.Issue.Fields.Project.Key) {
			continue
		}

		validFilter := true

		for _, field := range sub.Filters.Fields {
			// Broken filter, values must be provided
			if field.Values.Len() == 0 || field.Inclusion == "" {
				validFilter = false
				break
			}

			value := getIssueFieldValue(&jwh.Issue, field.Key)
			containsAny := value.ContainsAny(field.Values.Elems()...)
			containsAll := value.ContainsAll(field.Values.Elems()...)

			if (field.Inclusion == FILTER_INCLUDE_ANY && !containsAny) ||
				(field.Inclusion == FILTER_INCLUDE_ALL && !containsAll) ||
				(field.Inclusion == FILTER_EXCLUDE_ANY && containsAny) {
				validFilter = false
				break
			}
		}

		if !validFilter {
			continue
		}

		channelIds = channelIds.Add(sub.ChannelId)
	}

	return channelIds, nil
}

func (p *Plugin) getSubscriptions() (*Subscriptions, error) {
	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return nil, err
	}

	subKey := keyWithInstance(ji, JIRA_SUBSCRIPTIONS_KEY)
	data, appErr := p.API.KVGet(subKey)
	if appErr != nil {
		return nil, appErr
	}
	return SubscriptionsFromJson(data)
}

func (p *Plugin) getSubscriptionsForChannel(channelId string) ([]ChannelSubscription, error) {
	subs, err := p.getSubscriptions()
	if err != nil {
		return nil, err
	}

	channelSubscriptions := []ChannelSubscription{}
	for _, channelSubscriptionId := range subs.Channel.IdByChannelId[channelId].Elems() {
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
	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return err
	}

	subKey := keyWithInstance(ji, JIRA_SUBSCRIPTIONS_KEY)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
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
	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return err
	}

	subKey := keyWithInstance(ji, JIRA_SUBSCRIPTIONS_KEY)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		err = p.checkChannelSubscriptionNameUnique(newSubscription.ChannelId, newSubscription)
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

func (p *Plugin) checkChannelSubscriptionNameUnique(channelId string, subscription *ChannelSubscription) error {
	subs, err := p.getSubscriptionsForChannel(channelId)
	if err != nil {
		return err
	}

	for subID := range subs {
		if subs[subID].Name == subscription.Name && subs[subID].Id != subscription.Id {
			return fmt.Errorf("Subscription name, '%s', already exists. Please choose another name.", subs[subID].Name)
		}
	}
	return nil
}

func (p *Plugin) editChannelSubscription(modifiedSubscription *ChannelSubscription) error {
	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return err
	}

	subKey := keyWithInstance(ji, JIRA_SUBSCRIPTIONS_KEY)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		oldSub, ok := subs.Channel.ById[modifiedSubscription.Id]
		if !ok {
			return nil, errors.New("Existing subscription does not exist.")
		}

		err = p.checkChannelSubscriptionNameUnique(oldSub.ChannelId, modifiedSubscription)
		if err != nil {
			return nil, err
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

func inAllowedGroup(inGroups []*jira.UserGroup, allowedGroups []string) bool {
	for _, inGroup := range inGroups {
		for _, allowedGroup := range allowedGroups {
			if strings.TrimSpace(inGroup.Name) == strings.TrimSpace(allowedGroup) {
				return true
			}
		}
	}
	return false
}

// hasPermissionToManageSubscription checks if MM user has permission to manage subscriptions in given channel.
// returns nil if the user has permission and a descriptive error otherwise.
func (p *Plugin) hasPermissionToManageSubscription(userId, channelId string) error {
	cfg := p.getConfig()

	switch cfg.RolesAllowedToEditJiraSubscriptions {
	case "team_admin":
		if !p.API.HasPermissionToChannel(userId, channelId, model.PERMISSION_MANAGE_TEAM) {
			return errors.New("is not team admin")
		}
	case "channel_admin":
		channel, appErr := p.API.GetChannel(channelId)
		if appErr != nil {
			return errors.Wrap(appErr, "unable to get channel to check permission")
		}
		switch channel.Type {
		case model.CHANNEL_OPEN:
			if !p.API.HasPermissionToChannel(userId, channelId, model.PERMISSION_MANAGE_PUBLIC_CHANNEL_PROPERTIES) {
				return errors.New("is not channel admin")
			}
		case model.CHANNEL_PRIVATE:
			if !p.API.HasPermissionToChannel(userId, channelId, model.PERMISSION_MANAGE_PRIVATE_CHANNEL_PROPERTIES) {
				return errors.New("is not channel admin")
			}
		default:
			return errors.New("can only subscribe in public and private channels")
		}
	case "users":
	default:
		if !p.API.HasPermissionTo(userId, model.PERMISSION_MANAGE_SYSTEM) {
			return errors.New("is not system admin")
		}
	}

	if cfg.GroupsAllowedToEditJiraSubscriptions != "" {
		ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
		if err != nil {
			return errors.Wrap(err, "could not load jira instance")
		}

		jiraUser, err := p.userStore.LoadJIRAUser(ji, userId)
		if err != nil {
			return errors.Wrap(err, "could not load jira user")
		}

		client, err := ji.GetClient(jiraUser)
		if err != nil {
			return errors.Wrap(err, "could not get an authenticated Jira client")
		}

		groups, err := client.GetUserGroups(jiraUser)
		if err != nil {
			return errors.Wrap(err, "could not get jira user groups")
		}

		allowedGroups := strings.Split(cfg.GroupsAllowedToEditJiraSubscriptions, ",")
		allowedGroups = Map(allowedGroups, strings.TrimSpace)
		if !inAllowedGroup(groups, allowedGroups) {
			return errors.New("not in allowed jira user groups")
		}
	}

	return nil
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
		success = true
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
	if cfg.Secret == "" {
		return http.StatusForbidden, fmt.Errorf("JIRA plugin not configured correctly; must provide Secret")
	}

	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(cfg.Secret)) != 1 {
		return http.StatusForbidden, fmt.Errorf("Request URL: secret did not match")
	}

	bb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// If there is space in the queue, immediately return a 200; we will process the webhook event async.
	// If the queue is full, return a 503; we will not process that webhook event.
	select {
	case p.webhookQueue <- bb:
		return http.StatusOK, nil
	default:
		return http.StatusServiceUnavailable, nil
	}
}

func httpChannelCreateSubscription(p *Plugin, w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscription := ChannelSubscription{}
	if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 0 {
		return http.StatusBadRequest, fmt.Errorf("Channel subscription invalid")
	}

	if _, err := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId); err != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	if err := p.hasPermissionToManageSubscription(mattermostUserId, subscription.ChannelId); err != nil {
		return http.StatusForbidden, errors.Wrap(err, "you don't have permission to manage subscriptions")
	}

	if err := p.addChannelSubscription(&subscription); err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(&subscription)
	if _, err := w.Write(b); err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	post := &model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelId,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was added to this channel by %v", subscription.Name, jiraUser.DisplayName),
	}

	p.API.CreatePost(post)

	return http.StatusOK, nil
}

func httpChannelEditSubscription(p *Plugin, w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscription := ChannelSubscription{}
	if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 26 {
		return http.StatusBadRequest, fmt.Errorf("Channel subscription invalid")
	}

	if err := p.hasPermissionToManageSubscription(mattermostUserId, subscription.ChannelId); err != nil {
		return http.StatusForbidden, errors.Wrap(err, "you don't have permission to manage subscriptions")
	}

	if _, err := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId); err != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	if err := p.editChannelSubscription(&subscription); err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(&subscription)
	if _, err := w.Write(b); err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	post := &model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelId,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was updated by %v", subscription.Name, jiraUser.DisplayName),
	}

	p.API.CreatePost(post)

	return http.StatusOK, nil
}

func httpChannelDeleteSubscription(p *Plugin, w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscriptionId := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(subscriptionId) != 26 {
		return http.StatusBadRequest, errors.New("bad subscription id")
	}

	subscription, err := p.getChannelSubscription(subscriptionId)
	if err != nil {
		return http.StatusBadRequest, errors.Wrap(err, "bad subscription id")
	}

	if err := p.hasPermissionToManageSubscription(mattermostUserId, subscription.ChannelId); err != nil {
		return http.StatusForbidden, errors.Wrap(err, "you don't have permission to manage subscriptions")
	}

	if _, err := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId); err != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	if err := p.removeChannelSubscription(subscriptionId); err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "unable to remove channel subscription")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"status\": \"OK\"}"))

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	post := &model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelId,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was removed from this channel by %v", subscription.Name, jiraUser.DisplayName),
	}

	p.API.CreatePost(post)

	return http.StatusOK, nil
}

func httpChannelGetSubscriptions(p *Plugin, w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	channelId := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(channelId) != 26 {
		return http.StatusBadRequest, errors.New("bad channel id")
	}

	if _, err := p.API.GetChannelMember(channelId, mattermostUserId); err != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	if err := p.hasPermissionToManageSubscription(mattermostUserId, channelId); err != nil {
		return http.StatusForbidden, errors.Wrap(err, "you don't have permission to manage subscriptions")
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

	return http.StatusOK, nil
}

func httpChannelSubscriptions(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	switch r.Method {
	case http.MethodPost:
		return httpChannelCreateSubscription(p, w, r, mattermostUserId)
	case http.MethodDelete:
		return httpChannelDeleteSubscription(p, w, r, mattermostUserId)
	case http.MethodGet:
		return httpChannelGetSubscriptions(p, w, r, mattermostUserId)
	case http.MethodPut:
		return httpChannelEditSubscription(p, w, r, mattermostUserId)
	default:
		return http.StatusMethodNotAllowed, fmt.Errorf("Request: " + r.Method + " is not allowed.")
	}
}
