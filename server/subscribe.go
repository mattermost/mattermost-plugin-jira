// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
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
	FILTER_EMPTY       = "empty"

	MAX_SUBSCRIPTION_NAME_LENGTH = 100
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

	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return nil, err
	}

	subIds := subs.Channel.ById
	instType := ji.GetType()

	issue := &jwh.Issue
	webhookEvents := wh.Events()
	isCommentEvent := jwh.WebhookEvent == "comment_created" || jwh.WebhookEvent == "comment_updated" || jwh.WebhookEvent == "comment_deleted"

	if isCommentEvent && instType == "cloud" {
		// Jira Cloud comment event. We need to fetch issue data because it is not expanded in webhook payload.
		issue, err = p.getIssueDataForCloudWebhook(ji, issue.ID)
		if err != nil {
			return nil, err
		}
	}

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

		if !sub.Filters.IssueTypes.ContainsAny(issue.Fields.Type.ID) {
			continue
		}

		if !sub.Filters.Projects.ContainsAny(issue.Fields.Project.Key) {
			continue
		}

		validFilter := true

		for _, field := range sub.Filters.Fields {
			// Broken filter, values must be provided
			if field.Inclusion == "" || (field.Values.Len() == 0 && field.Inclusion != FILTER_EMPTY) {
				validFilter = false
				break
			}

			value := getIssueFieldValue(issue, field.Key)
			containsAny := value.ContainsAny(field.Values.Elems()...)
			containsAll := value.ContainsAll(field.Values.Elems()...)

			if (field.Inclusion == FILTER_INCLUDE_ANY && !containsAny) ||
				(field.Inclusion == FILTER_INCLUDE_ALL && !containsAll) ||
				(field.Inclusion == FILTER_EXCLUDE_ANY && containsAny) ||
				(field.Inclusion == FILTER_EMPTY && value.Len() > 0) {
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

		err = p.validateSubscription(newSubscription)
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

func (p *Plugin) validateSubscription(subscription *ChannelSubscription) error {
	if len(subscription.Name) == 0 {
		return errors.New("Please provide a name for the subscription.")
	}

	if len(subscription.Name) > MAX_SUBSCRIPTION_NAME_LENGTH {
		return fmt.Errorf("Please provide a name less than %d characters.", MAX_SUBSCRIPTION_NAME_LENGTH)
	}

	channelId := subscription.ChannelId
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

		err = p.validateSubscription(modifiedSubscription)
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

type SubsGroupedByTeam struct {
	TeamId               string
	TeamName             string
	SubsGroupedByChannel []SubsGroupedByChannel
}

type SubsGroupedByChannel struct {
	ChannelId string
	SubIds    []string
}

func (p *Plugin) listChannelSubscriptions(teamId string) (string, error) {
	subs, err := p.getSubscriptions()
	if err != nil {
		return "", err
	}

	sortedSubs, err := p.getSortedSubscriptions()
	if err != nil {
		return "", err
	}

	rows := []string{}

	if sortedSubs == nil {
		rows = append(rows, fmt.Sprintf("There are currently no channels subcriptions to Jira notifications. To add a subscription, navigate to a channel and type `/jira subscribe`\n"))
		return strings.Join(rows, "\n"), nil
	}
	rows = append(rows, fmt.Sprintf("The following channels have subscribed to Jira notifications. To modify a subscription, navigate to the channel and type `/jira subscribe`"))

	for _, teamSubs := range sortedSubs {

		// create header for each Team, DM and GM channels
		rows = append(rows, fmt.Sprintf("\n#### %s", teamSubs.TeamName))

		for _, grouped := range teamSubs.SubsGroupedByChannel {

			channel, appErr := p.API.GetChannel(grouped.ChannelId)
			if appErr != nil {
				return "", errors.New("Failed to get channel")
			}

			// only print channel name once for all subscriptions
			channelRow := fmt.Sprintf("* **%s** (%d):", channel.Name, len(grouped.SubIds))
			if teamId == teamSubs.TeamId {
				// only link the channels on the current team
				channelRow = fmt.Sprintf("* **~%s** (%d):", channel.Name, len(grouped.SubIds))
			}
			rows = append(rows, channelRow)

			for _, subId := range grouped.SubIds {
				sub := subs.Channel.ById[subId]

				subName := "(No Name)"
				if sub.Name != "" {
					subName = sub.Name
				}
				rows = append(rows, fmt.Sprintf("  * %s - %s", sub.Filters.Projects.Elems()[0], subName))

			}
		}
	}

	return strings.Join(rows, "\n"), nil
}

func (p *Plugin) getSortedSubscriptions() ([]SubsGroupedByTeam, error) {
	subs, err := p.getSubscriptions()
	if err != nil {
		return nil, err
	}

	subsMap := make(map[string][]SubsGroupedByChannel)
	teamMap := make(map[string]string)

	var teams []model.Team
	var dmSubsIds []SubsGroupedByChannel

	// get teams from subscriptions
	for channelID, subIDs := range subs.Channel.IdByChannelId {

		// channel does not have any subIDs.
		if len(subIDs) == 0 {
			continue
		}

		channel, appErr := p.API.GetChannel(channelID)
		if appErr != nil {
			return nil, errors.New("Failed to get channel")
		}

		var channelSubIds []string
		for subID := range subIDs {
			channelSubIds = append(channelSubIds, subID)
		}

		grouped := SubsGroupedByChannel{
			ChannelId: channelID,
			SubIds:    channelSubIds,
		}
		// for DMs and GMs, save to array and go to next team
		if channel.TeamId == "" {
			dmSubsIds = append(dmSubsIds, grouped)
			continue
		}

		// teamMap used to determine if already have the team saved
		_, ok := teamMap[channel.TeamId]
		if !ok {
			team, _ := p.API.GetTeam(channel.TeamId)
			teams = append(teams, *team)
			teamMap[channel.TeamId] = team.DisplayName
		}

		// only save non-DM and non-GM subs to the map
		subsMap[channel.TeamId] = append(subsMap[channel.TeamId], grouped)

	}

	var teamSubs []SubsGroupedByTeam

	// Closures that order the Teams structure.
	displayName := func(p1, p2 *model.Team) bool {
		return p1.DisplayName < p2.DisplayName
	}

	// Sort the teams by the various criteria.
	By(displayName).Sort(teams)

	for _, teamId := range teams {
		teamData := SubsGroupedByTeam{
			TeamId:               teamId.Id,
			TeamName:             teamId.DisplayName,
			SubsGroupedByChannel: subsMap[teamId.Id],
		}
		teamSubs = append(teamSubs, teamData)
	}

	// save all DM and GM channels under a generic teamName
	if len(dmSubsIds) != 0 {
		teamData := SubsGroupedByTeam{
			TeamId:               "",
			TeamName:             "Group and Direct Messages",
			SubsGroupedByChannel: dmSubsIds,
		}
		teamSubs = append(teamSubs, teamData)
	}

	return teamSubs, nil
}

type By func(p1, p2 *model.Team) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(teams []model.Team) {
	ps := &teamSorter{
		teams: teams,
		by:    by,
	}
	sort.Sort(ps)
}

type teamSorter struct {
	teams []model.Team
	by    func(p1, p2 *model.Team) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *teamSorter) Len() int {
	return len(s.teams)
}

// Swap is part of sort.Interface.
func (s *teamSorter) Swap(i, j int) {
	s.teams[i], s.teams[j] = s.teams[j], s.teams[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *teamSorter) Less(i, j int) bool {
	return s.by(&s.teams[i], &s.teams[j])
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
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 0 {
		return http.StatusBadRequest, fmt.Errorf("Channel subscription invalid")
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	err = p.hasPermissionToManageSubscription(mattermostUserId, subscription.ChannelId)
	if err != nil {
		return http.StatusForbidden, errors.Wrap(err, "you don't have permission to manage subscriptions")
	}

	err = p.addChannelSubscription(&subscription)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(&subscription)
	_, err = w.Write(b)
	if err != nil {
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
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 26 {
		return http.StatusBadRequest, fmt.Errorf("Channel subscription invalid")
	}

	err = p.hasPermissionToManageSubscription(mattermostUserId, subscription.ChannelId)
	if err != nil {
		return http.StatusForbidden, errors.Wrap(err, "you don't have permission to manage subscriptions")
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	err = p.editChannelSubscription(&subscription)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(&subscription)
	_, err = w.Write(b)
	if err != nil {
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

	err = p.hasPermissionToManageSubscription(mattermostUserId, subscription.ChannelId)
	if err != nil {
		return http.StatusForbidden, errors.Wrap(err, "you don't have permission to manage subscriptions")
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	err = p.removeChannelSubscription(subscriptionId)
	if err != nil {
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
