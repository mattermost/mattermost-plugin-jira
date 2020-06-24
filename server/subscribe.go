// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v5/model"
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
	Id         string              `json:"id"`
	ChannelId  string              `json:"channel_id"`
	Filters    SubscriptionFilters `json:"filters"`
	Name       string              `json:"name"`
	InstanceID types.ID            `json:"instance_id"`
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

func SubscriptionsFromJson(bytes []byte, instanceID types.ID) (*Subscriptions, error) {
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

	for _, sub := range subs.Channel.ById {
		sub.InstanceID = instanceID
	}

	return subs, nil
}

func (p *Plugin) getUserID() string {
	return p.getConfig().botUserID
}

func (p *Plugin) matchesSubsciptionFilters(wh *webhook, filters SubscriptionFilters) bool {
	webhookEvents := wh.Events()
	foundEvent := false
	eventTypes := filters.Events
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
		return false
	}

	if filters.IssueTypes.Len() != 0 && !filters.IssueTypes.ContainsAny(wh.JiraWebhook.Issue.Fields.Type.ID) {
		return false
	}

	if filters.Projects.Len() != 0 && !filters.Projects.ContainsAny(wh.JiraWebhook.Issue.Fields.Project.Key) {
		return false
	}

	validFilter := true

	for _, field := range filters.Fields {
		// Broken filter, values must be provided
		if field.Inclusion == "" || (field.Values.Len() == 0 && field.Inclusion != FILTER_EMPTY) {
			validFilter = false
			break
		}

		value := getIssueFieldValue(&wh.JiraWebhook.Issue, field.Key)
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
		return false
	}

	return true
}

func (p *Plugin) getChannelsSubscribed(wh *webhook, instanceID types.ID) (StringSet, error) {
	subs, err := p.getSubscriptions(instanceID)
	if err != nil {
		return nil, err
	}

	channelIds := NewStringSet()
	subIds := subs.Channel.ById
	for _, sub := range subIds {
		if p.matchesSubsciptionFilters(wh, sub.Filters) {
			channelIds = channelIds.Add(sub.ChannelId)
		}
	}

	return channelIds, nil
}

func (p *Plugin) getSubscriptions(instanceID types.ID) (*Subscriptions, error) {
	subKey := keyWithInstanceID(instanceID, JIRA_SUBSCRIPTIONS_KEY)
	data, appErr := p.API.KVGet(subKey)
	if appErr != nil {
		return nil, appErr
	}
	return SubscriptionsFromJson(data, instanceID)
}

func (p *Plugin) getSubscriptionsForChannel(instanceID types.ID, channelId string) ([]ChannelSubscription, error) {
	subs, err := p.getSubscriptions(instanceID)
	if err != nil {
		return nil, err
	}

	channelSubscriptions := []ChannelSubscription{}
	for _, channelSubscriptionId := range subs.Channel.IdByChannelId[channelId].Elems() {
		channelSubscriptions = append(channelSubscriptions, subs.Channel.ById[channelSubscriptionId])
	}

	return channelSubscriptions, nil
}

func (p *Plugin) getChannelSubscription(instanceID types.ID, subscriptionId string) (*ChannelSubscription, error) {
	subs, err := p.getSubscriptions(instanceID)
	if err != nil {
		return nil, err
	}

	subscription, ok := subs.Channel.ById[subscriptionId]
	if !ok {
		return nil, errors.New("could not find subscription")
	}

	return &subscription, nil
}

func (p *Plugin) removeChannelSubscription(instanceID types.ID, subscriptionId string) error {
	subKey := keyWithInstanceID(instanceID, JIRA_SUBSCRIPTIONS_KEY)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes, instanceID)
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

func (p *Plugin) addChannelSubscription(instanceID types.ID, newSubscription *ChannelSubscription, client Client) error {
	subKey := keyWithInstanceID(instanceID, JIRA_SUBSCRIPTIONS_KEY)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes, instanceID)
		if err != nil {
			return nil, err
		}

		err = p.validateSubscription(instanceID, newSubscription, client)
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

func (p *Plugin) validateSubscription(instanceID types.ID, subscription *ChannelSubscription, client Client) error {
	if len(subscription.Name) == 0 {
		return errors.New("Please provide a name for the subscription.")
	}

	if len(subscription.Name) > MAX_SUBSCRIPTION_NAME_LENGTH {
		return errors.Errorf("Please provide a name less than %d characters.", MAX_SUBSCRIPTION_NAME_LENGTH)
	}

	if len(subscription.Filters.Events) == 0 {
		return errors.New("Please provide at least one event type.")
	}

	if len(subscription.Filters.IssueTypes) == 0 {
		return errors.New("Please provide at least one issue type.")
	}

	if (len(subscription.Filters.Projects)) == 0 {
		return errors.New("Please provide a project identifier.")
	}

	channelId := subscription.ChannelId
	subs, err := p.getSubscriptionsForChannel(instanceID, channelId)
	if err != nil {
		return err
	}

	for subID := range subs {
		if subs[subID].Name == subscription.Name && subs[subID].Id != subscription.Id {
			return errors.Errorf("Subscription name, '%s', already exists. Please choose another name.", subs[subID].Name)
		}
	}

	projectKey := subscription.Filters.Projects.Elems()[0]
	_, err = client.GetProject(projectKey)
	if err != nil {
		return errors.WithMessagef(err, "failed to get project %q", projectKey)
	}

	return nil
}

func (p *Plugin) editChannelSubscription(instanceID types.ID, modifiedSubscription *ChannelSubscription, client Client) error {
	subKey := keyWithInstanceID(instanceID, JIRA_SUBSCRIPTIONS_KEY)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes, instanceID)
		if err != nil {
			return nil, err
		}

		oldSub, ok := subs.Channel.ById[modifiedSubscription.Id]
		if !ok {
			return nil, errors.New("Existing subscription does not exist.")
		}

		err = p.validateSubscription(instanceID, modifiedSubscription, client)
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

func (p *Plugin) listChannelSubscriptions(instanceID types.ID, teamId string) (string, error) {
	subs, err := p.getSubscriptions(instanceID)
	if err != nil {
		return "", err
	}

	sortedSubs, err := p.getSortedSubscriptions(instanceID)
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

func (p *Plugin) getSortedSubscriptions(instanceID types.ID) ([]SubsGroupedByTeam, error) {
	subs, err := p.getSubscriptions(instanceID)
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
func (p *Plugin) hasPermissionToManageSubscription(instanceID types.ID, userId, channelId string) error {
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
		instance, err := p.instanceStore.LoadInstance(instanceID)
		if err != nil {
			return errors.Wrap(err, "could not load jira instance")
		}

		c, err := p.userStore.LoadConnection(instance.GetID(), types.ID(userId))
		if err != nil {
			return errors.Wrap(err, "could not load jira user")
		}

		client, err := instance.GetClient(c)
		if err != nil {
			return errors.Wrap(err, "could not get an authenticated Jira client")
		}

		groups, err := client.GetUserGroups(c)
		if err != nil {
			return errors.Wrap(err, "could not get jira user groups")
		}

		allowedGroups := strings.Split(cfg.GroupsAllowedToEditJiraSubscriptions, ",")
		allowedGroups = utils.Map(allowedGroups, strings.TrimSpace)
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

	var (
		retryLimit     = 5
		retryWait      = 30 * time.Millisecond
		success        = false
		currentAttempt = 0
	)
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

		if currentAttempt == 0 && bytes.Equal(initialBytes, newValue) {
			return nil
		}

		currentAttempt++
		if currentAttempt >= retryLimit {
			return errors.New("reached write attempt limit")
		}

		time.Sleep(retryWait)
	}

	return nil
}

func (p *Plugin) httpSubscribeWebhook(w http.ResponseWriter, r *http.Request, instanceID types.ID) (status int, err error) {
	conf := p.getConfig()
	size := utils.ByteSize(0)
	start := time.Now()
	defer func() {
		if conf.stats != nil {
			conf.stats.EnsureEndpoint("jira/subscribe/response").Record(size, 0, time.Since(start), err != nil, false)
		}
	}()

	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			fmt.Errorf("Request: "+r.Method+" is not allowed, must be POST"))
	}
	if conf.Secret == "" {
		return respondErr(w, http.StatusForbidden,
			fmt.Errorf("JIRA plugin not configured correctly; must provide Secret"))
	}
	status, err = verifyHTTPSecret(conf.Secret, r.FormValue("secret"))
	if err != nil {
		return respondErr(w, status, err)
	}

	bb, err := ioutil.ReadAll(r.Body)
	size = utils.ByteSize(len(bb))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// If there is space in the queue, immediately return a 200; we will process the webhook event async.
	// If the queue is full, return a 503; we will not process that webhook event.
	select {
	case p.webhookQueue <- &webhookMessage{
		InstanceID: instanceID,
		Data:       bb,
	}:
		return http.StatusOK, nil
	default:
		return respondErr(w, http.StatusServiceUnavailable, nil)
	}
}

func (p *Plugin) httpChannelCreateSubscription(w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 0 {
		return respondErr(w, http.StatusBadRequest,
			fmt.Errorf("Channel subscription invalid"))
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("Not a member of the channel specified"))
	}

	err = p.hasPermissionToManageSubscription(subscription.InstanceID, mattermostUserId, subscription.ChannelId)
	if err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	client, _, connection, err := p.getClient(subscription.InstanceID, types.ID(mattermostUserId))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	err = p.addChannelSubscription(subscription.InstanceID, &subscription, client)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	projectKey := ""
	if subscription.Filters.Projects.Len() == 1 {
		projectKey = subscription.Filters.Projects.Elems()[0]
	}
	p.UpdateUserDefaults(types.ID(mattermostUserId), subscription.InstanceID, projectKey)

	code, err := respondJSON(w, &subscription)
	if err != nil {
		return code, err
	}

	p.API.CreatePost(&model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelId,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was added to this channel by %v", subscription.Name, connection.DisplayName),
	})
	return http.StatusOK, nil
}

func (p *Plugin) httpChannelEditSubscription(w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 26 {
		return respondErr(w, http.StatusBadRequest,
			fmt.Errorf("Channel subscription invalid"))
	}

	err = p.hasPermissionToManageSubscription(subscription.InstanceID, mattermostUserId, subscription.ChannelId)
	if err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("Not a member of the channel specified"))
	}

	client, _, connection, err := p.getClient(subscription.InstanceID, types.ID(mattermostUserId))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	err = p.editChannelSubscription(subscription.InstanceID, &subscription, client)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	projectKey := ""
	if subscription.Filters.Projects.Len() == 1 {
		projectKey = subscription.Filters.Projects.Elems()[0]
	}
	p.UpdateUserDefaults(types.ID(mattermostUserId), subscription.InstanceID, projectKey)

	code, err := respondJSON(w, &subscription)
	if err != nil {
		return code, err
	}

	p.API.CreatePost(&model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelId,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was updated by %v", subscription.Name, connection.DisplayName),
	})
	return http.StatusOK, nil
}

func (p *Plugin) httpChannelDeleteSubscription(w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	subscriptionId := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(subscriptionId) != 26 {
		return respondErr(w, http.StatusBadRequest,
			errors.New("bad subscription id"))
	}

	instanceID := types.ID(r.FormValue("instance_id"))
	subscription, err := p.getChannelSubscription(instanceID, subscriptionId)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.Wrap(err, "bad subscription id"))
	}

	err = p.hasPermissionToManageSubscription(instanceID, mattermostUserId, subscription.ChannelId)
	if err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("Not a member of the channel specified"))
	}

	err = p.removeChannelSubscription(instanceID, subscriptionId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.Wrap(err, "unable to remove channel subscription"))
	}

	code, err := respondJSON(w, map[string]interface{}{"status": "OK"})
	if err != nil {
		return code, err
	}

	connection, err := p.userStore.LoadConnection(instanceID, types.ID(mattermostUserId))
	if err != nil {
		return http.StatusInternalServerError, err
	}
	p.API.CreatePost(&model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelId,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was removed from this channel by %v", subscription.Name, connection.DisplayName),
	})
	return http.StatusOK, nil
}

func (p *Plugin) httpChannelGetSubscriptions(w http.ResponseWriter, r *http.Request, mattermostUserId string) (int, error) {
	channelId := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(channelId) != 26 {
		return respondErr(w, http.StatusBadRequest,
			errors.New("bad channel id"))
	}
	instanceID := types.ID(r.FormValue("instance_id"))

	if _, appErr := p.API.GetChannelMember(channelId, mattermostUserId); appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("Not a member of the channel specified"))
	}

	if err := p.hasPermissionToManageSubscription(instanceID, mattermostUserId, channelId); err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	subscriptions, err := p.getSubscriptionsForChannel(instanceID, channelId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.Wrap(err, "unable to get channel subscriptions"))
	}

	return respondJSON(w, subscriptions)
}

func (p *Plugin) httpChannelSubscriptions(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	switch r.Method {
	case http.MethodPost:
		return p.httpChannelCreateSubscription(w, r, mattermostUserId)
	case http.MethodDelete:
		return p.httpChannelDeleteSubscription(w, r, mattermostUserId)
	case http.MethodGet:
		return p.httpChannelGetSubscriptions(w, r, mattermostUserId)
	case http.MethodPut:
		return p.httpChannelEditSubscription(w, r, mattermostUserId)
	default:
		return respondErr(w, http.StatusMethodNotAllowed, fmt.Errorf("Request: "+r.Method+" is not allowed."))
	}
}
