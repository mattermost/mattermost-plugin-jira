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

	"github.com/andygrunwald/go-jira"
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

func (p *Plugin) getUserID() string {
	return p.getConfig().botUserID
}

func (p *Plugin) getChannelsSubscribed(jwh *JiraWebhook) ([]string, error) {
	subs, err := p.getSubscriptions()
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

	switch cfg.SubscriptionEditorsMattermostRoles {
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

	if cfg.SubscriptionEditorsJiraGroups != "" {
		ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
		if err != nil {
			return errors.Wrap(err, "could not load jira instance")
		}

		jiraUser, err := p.userStore.LoadJIRAUser(ji, userId)
		if err != nil {
			return errors.Wrap(err, "could not load jira user")
		}

		jiraClient, err := ji.GetJIRAClient(jiraUser)
		if err != nil {
			return errors.Wrap(err, "could not get jira client")
		}

		req, err := jiraClient.NewRequest("GET", "rest/api/3/user/groups?key="+jiraUser.Key, nil)
		if err != nil {
			return errors.Wrap(err, "error creating request")
		}

		var groups []*jira.UserGroup
		resp, err := jiraClient.Do(req, &groups)
		if err != nil {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			return errors.Wrap(err, "error in request to get user groups, body:"+string(body))
		}

		allowedGroups := strings.Split(cfg.SubscriptionEditorsJiraGroups, ",")
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

	wh, jwh, err := ParseWebhook(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	channelIds, err := p.getChannelsSubscribed(jwh)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	botUserId := p.getUserID()

	for _, channelId := range channelIds {
		if _, status, err1 := wh.PostToChannel(p, channelId, botUserId); err1 != nil {
			return status, err1
		}
	}

	_, status, err := wh.PostNotifications(p)
	if err != nil {
		return status, err
	}
	return http.StatusOK, nil
}

func httpChannelCreateSubscription(p *Plugin, w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
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
	w.Write([]byte("{\"status\": \"OK\"}"))

	return http.StatusOK, nil
}

func httpChannelEditSubscription(p *Plugin, w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
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
	w.Write([]byte("{\"status\": \"OK\"}"))

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
