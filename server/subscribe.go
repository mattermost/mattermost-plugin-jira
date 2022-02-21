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

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	JiraSubscriptionsKey = "jirasub"
	FilterIncludeAny     = "include_any"
	FilterIncludeAll     = "include_all"
	FilterExcludeAny     = "exclude_any"
	FilterEmpty          = "empty"

	MaxSubscriptionNameLength = 100
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
	ID         string              `json:"id"`
	ChannelID  string              `json:"channel_id"`
	Filters    SubscriptionFilters `json:"filters"`
	Name       string              `json:"name"`
	InstanceID types.ID            `json:"instance_id"`
}

type ChannelSubscriptions struct {
	ByID          map[string]ChannelSubscription `json:"by_id"`
	IDByChannelID map[string]StringSet           `json:"id_by_channel_id"`
	IDByEvent     map[string]StringSet           `json:"id_by_event"`
}

func NewChannelSubscriptions() *ChannelSubscriptions {
	return &ChannelSubscriptions{
		ByID:          map[string]ChannelSubscription{},
		IDByChannelID: map[string]StringSet{},
		IDByEvent:     map[string]StringSet{},
	}
}

func (s *ChannelSubscriptions) remove(sub *ChannelSubscription) {
	delete(s.ByID, sub.ID)

	s.IDByChannelID[sub.ChannelID] = s.IDByChannelID[sub.ChannelID].Subtract(sub.ID)

	for _, event := range sub.Filters.Events.Elems() {
		s.IDByEvent[event] = s.IDByEvent[event].Subtract(sub.ID)
	}
}

func (s *ChannelSubscriptions) add(newSubscription *ChannelSubscription) {
	s.ByID[newSubscription.ID] = *newSubscription
	s.IDByChannelID[newSubscription.ChannelID] = s.IDByChannelID[newSubscription.ChannelID].Add(newSubscription.ID)
	for _, event := range newSubscription.Filters.Events.Elems() {
		s.IDByEvent[event] = s.IDByEvent[event].Add(newSubscription.ID)
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

func SubscriptionsFromJSON(bytes []byte, instanceID types.ID) (*Subscriptions, error) {
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

	// Backfill instance id's for old subscriptions
	for subID, sub := range subs.Channel.ByID {
		sub.InstanceID = instanceID
		subs.Channel.ByID[subID] = sub
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
			if strings.HasPrefix(eventType, "event_updated") || strings.HasSuffix(eventType, "comment") {
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
		if field.Inclusion == "" || (field.Values.Len() == 0 && field.Inclusion != FilterEmpty) {
			validFilter = false
			break
		}

		value := getIssueFieldValue(&wh.JiraWebhook.Issue, field.Key)
		containsAny := value.ContainsAny(field.Values.Elems()...)
		containsAll := value.ContainsAll(field.Values.Elems()...)

		if (field.Inclusion == FilterIncludeAny && !containsAny) ||
			(field.Inclusion == FilterIncludeAll && !containsAll) ||
			(field.Inclusion == FilterExcludeAny && containsAny) ||
			(field.Inclusion == FilterEmpty && value.Len() > 0) {
			validFilter = false
			break
		}
	}

	return validFilter
}

func (p *Plugin) getChannelsSubscribed(wh *webhook, instanceID types.ID) ([]ChannelSubscription, error) {
	subs, err := p.getSubscriptions(instanceID)
	if err != nil {
		return nil, err
	}

	var channelSubscriptions []ChannelSubscription
	subIds := subs.Channel.ByID
	for _, sub := range subIds {
		if p.matchesSubsciptionFilters(wh, sub.Filters) {
			channelSubscriptions = append(channelSubscriptions, sub)
		}
	}

	return channelSubscriptions, nil
}

func (p *Plugin) getSubscriptions(instanceID types.ID) (*Subscriptions, error) {
	subKey := keyWithInstanceID(instanceID, JiraSubscriptionsKey)
	data, appErr := p.API.KVGet(subKey)
	if appErr != nil {
		return nil, appErr
	}
	return SubscriptionsFromJSON(data, instanceID)
}

func (p *Plugin) getSubscriptionsForChannel(instanceID types.ID, channelID string) ([]ChannelSubscription, error) {
	subs, err := p.getSubscriptions(instanceID)
	if err != nil {
		return nil, err
	}

	channelSubscriptions := []ChannelSubscription{}
	for _, channelSubscriptionID := range subs.Channel.IDByChannelID[channelID].Elems() {
		channelSubscriptions = append(channelSubscriptions, subs.Channel.ByID[channelSubscriptionID])
	}

	return channelSubscriptions, nil
}

func (p *Plugin) getChannelSubscription(instanceID types.ID, subscriptionID string) (*ChannelSubscription, error) {
	subs, err := p.getSubscriptions(instanceID)
	if err != nil {
		return nil, err
	}

	subscription, ok := subs.Channel.ByID[subscriptionID]
	if !ok {
		return nil, errors.New("could not find subscription")
	}

	return &subscription, nil
}

func (p *Plugin) removeChannelSubscription(instanceID types.ID, subscriptionID string) error {
	subKey := keyWithInstanceID(instanceID, JiraSubscriptionsKey)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJSON(initialBytes, instanceID)
		if err != nil {
			return nil, err
		}

		subscription, ok := subs.Channel.ByID[subscriptionID]
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
	subKey := keyWithInstanceID(instanceID, JiraSubscriptionsKey)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJSON(initialBytes, instanceID)
		if err != nil {
			return nil, err
		}

		err = p.validateSubscription(instanceID, newSubscription, client)
		if err != nil {
			return nil, err
		}

		newSubscription.ID = model.NewId()
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
		return errors.New("please provide a name for the subscription")
	}

	if len(subscription.Name) > MaxSubscriptionNameLength {
		return errors.Errorf("please provide a name less than %d characters", MaxSubscriptionNameLength)
	}

	if len(subscription.Filters.Events) == 0 {
		return errors.New("please provide at least one event type")
	}

	if len(subscription.Filters.IssueTypes) == 0 {
		return errors.New("please provide at least one issue type")
	}

	if (len(subscription.Filters.Projects)) == 0 {
		return errors.New("please provide a project identifier")
	}

	channelID := subscription.ChannelID
	subs, err := p.getSubscriptionsForChannel(instanceID, channelID)
	if err != nil {
		return err
	}

	for subID := range subs {
		if subs[subID].Name == subscription.Name && subs[subID].ID != subscription.ID {
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
	subKey := keyWithInstanceID(instanceID, JiraSubscriptionsKey)
	return p.atomicModify(subKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJSON(initialBytes, instanceID)
		if err != nil {
			return nil, err
		}

		oldSub, ok := subs.Channel.ByID[modifiedSubscription.ID]
		if !ok {
			return nil, errors.New("existing subscription does not exist")
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

type InstanceSubMap map[types.ID][]string
type ChannelSubMap map[string]InstanceSubMap
type TeamSubsMap map[string]ChannelSubMap

type SubsGroupedByTeam struct {
	TeamID   string
	TeamName string
	Subs     TeamSubsMap
}

type SubsGroupedByChannel struct {
	ChannelID  string
	NumberSubs int
	SubIds     []string
}

func (p *Plugin) listChannelSubscriptions(instanceID types.ID, teamID string) (string, error) {
	sortedSubs, err := p.getSortedSubscriptions(instanceID)
	if err != nil {
		return "", err
	}

	rows := []string{}

	if sortedSubs == nil {
		rows = append(rows, "There are currently no channels subcriptions to Jira notifications. To add a subscription, navigate to a channel and type `/jira subscribe edit`\n")
		return strings.Join(rows, "\n"), nil
	}
	rows = append(rows, "The following channels have subscribed to Jira notifications. To modify a subscription, navigate to the channel and type `/jira subscribe edit`")

	for _, teamSubs := range sortedSubs {
		// create header for each Team, DM and GM channels
		rows = append(rows, fmt.Sprintf("\n#### %s", teamSubs.TeamName))

		for channelID, channelGroup := range teamSubs.Subs[teamSubs.TeamID] {
			channel, appErr := p.API.GetChannel(channelID)
			if appErr != nil {
				return "", errors.New("failed to get channel")
			}

			// only print channel name once for all subscriptions
			channelRow := fmt.Sprintf("* **%s** (%d):", channel.Name, p.getNumSubsForChannel(channelGroup))
			if teamID == teamSubs.TeamID {
				// only link the channels on the current team
				channelRow = fmt.Sprintf("* **~%s** (%d):", channel.Name, p.getNumSubsForChannel(channelGroup))
			}
			rows = append(rows, channelRow)

			for instanceID, subsIDs := range channelGroup {
				subs, err := p.getSubscriptions(instanceID)
				if err != nil {
					return "", errors.New("failed to get subs")
				}
				rows = append(rows, fmt.Sprintf("\t* (%d) %s", len(subsIDs), instanceID))

				for _, subID := range subsIDs {
					sub := subs.Channel.ByID[subID]
					subName := "(No Name)"
					if sub.Name != "" {
						subName = sub.Name
					}
					rows = append(rows, fmt.Sprintf("\t\t* %s - %s", sub.Filters.Projects.Elems()[0], subName))
				}
			}
		}
	}

	return strings.Join(rows, "\n"), nil
}

func (p *Plugin) getNumSubsForChannel(channelGroup InstanceSubMap) int {
	totalSubs := 0
	for _, subsIDs := range channelGroup {
		totalSubs += len(subsIDs)
	}
	return totalSubs
}

func (p *Plugin) getSortedSubscriptions(instanceID types.ID) ([]SubsGroupedByTeam, error) {
	var instanceSubs []*Subscriptions
	if instanceID != "" {
		subs, err := p.getSubscriptions(instanceID)
		if err != nil {
			return nil, err
		}
		instanceSubs = append(instanceSubs, subs)
	} else {
		instances, err := p.instanceStore.LoadInstances()
		if err != nil {
			return nil, err
		}

		for _, instanceID := range instances.IDs() {
			subs, err := p.getSubscriptions(instanceID)
			if err != nil {
				return nil, err
			}
			instanceSubs = append(instanceSubs, subs)
		}
	}

	subsMap := make(TeamSubsMap)
	teamDisplayNameMap := make(map[string]string)

	var teams []model.Team
	var dmSubsIds []SubsGroupedByChannel

	for _, subs := range instanceSubs {
		// get teams from subscriptions
		for channelID, subIDs := range subs.Channel.IDByChannelID {
			// channel does not have any subIDs.
			if len(subIDs) == 0 {
				continue
			}

			channel, appErr := p.API.GetChannel(channelID)
			if appErr != nil {
				p.API.LogDebug("getSortedSubscriptions: failed to get channel.", "channelID", channelID, "error", appErr)
				continue
			}

			if subsMap[channel.TeamId] == nil {
				subsMap[channel.TeamId] = make(ChannelSubMap)
			}

			if subsMap[channel.TeamId][channelID] == nil {
				subsMap[channel.TeamId][channelID] = make(InstanceSubMap)
			}

			var channelSubIds []string
			for subID := range subIDs {
				channelSubIds = append(channelSubIds, subID)
				instanceID := subs.Channel.ByID[subID].InstanceID
				subsMap[channel.TeamId][channelID][instanceID] = append(subsMap[channel.TeamId][channelID][instanceID], subID)
			}

			grouped := SubsGroupedByChannel{
				ChannelID:  channelID,
				SubIds:     channelSubIds,
				NumberSubs: len(channelSubIds),
			}
			// for DMs and GMs, save to array and go to next team
			if channel.TeamId == "" {
				dmSubsIds = append(dmSubsIds, grouped)
				continue
			}

			// teamMap used to determine if already have the team saved
			_, ok := teamDisplayNameMap[channel.TeamId]
			if !ok {
				team, _ := p.API.GetTeam(channel.TeamId)
				teams = append(teams, *team)
				teamDisplayNameMap[channel.TeamId] = team.DisplayName
			}
		}
	}

	var teamSubs []SubsGroupedByTeam

	// Closures that order the Teams structure.
	displayName := func(p1, p2 *model.Team) bool {
		return p1.DisplayName < p2.DisplayName
	}

	// Sort the teams by the various criteria.
	By(displayName).Sort(teams)

	for _, teamID := range teams {
		teamData := SubsGroupedByTeam{
			TeamID:   teamID.Id,
			TeamName: teamID.DisplayName,
			Subs:     subsMap,
		}
		teamSubs = append(teamSubs, teamData)
	}

	// save all DM and GM channels under a generic teamName
	if len(dmSubsIds) != 0 {
		teamData := SubsGroupedByTeam{
			TeamID:   "",
			TeamName: "Group and Direct Messages",
			Subs:     subsMap,
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
func (p *Plugin) hasPermissionToManageSubscription(instanceID types.ID, userID, channelID string) error {
	cfg := p.getConfig()

	switch cfg.RolesAllowedToEditJiraSubscriptions {
	case "team_admin":
		if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionManageTeam) {
			return errors.New("is not team admin")
		}
	case "channel_admin":
		channel, appErr := p.API.GetChannel(channelID)
		if appErr != nil {
			return errors.Wrap(appErr, "unable to get channel to check permission")
		}
		switch channel.Type {
		case model.ChannelTypeOpen:
			if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionManagePublicChannelProperties) {
				return errors.New("is not channel admin")
			}
		case model.ChannelTypePrivate:
			if !p.API.HasPermissionToChannel(userID, channelID, model.PermissionManagePrivateChannelProperties) {
				return errors.New("is not channel admin")
			}
		default:
			return errors.New("can only subscribe in public and private channels")
		}
	case "users":
	default:
		if !p.API.HasPermissionTo(userID, model.PermissionManageSystem) {
			return errors.New("is not system admin")
		}
	}

	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return errors.Wrap(err, "could not load jira instance")
	}

	c, err := p.userStore.LoadConnection(instance.GetID(), types.ID(userID))
	if err != nil {
		return errors.Wrap(err, "could not load jira user")
	}

	if !instance.Common().IsV2Legacy || cfg.GroupsAllowedToEditJiraSubscriptions == "" {
		return nil
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

	return nil
}

func (p *Plugin) atomicModify(key string, modify func(initialValue []byte) ([]byte, error)) error {
	readModify := func() ([]byte, []byte, error) {
		initialBytes, appErr := p.API.KVGet(key)
		if appErr != nil {
			return nil, nil, errors.Wrap(appErr, "unable to read initial value")
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

	if conf.EnableWebhookEventLogging {
		p.API.LogDebug("Webhook Event Log", "event", string(bb))
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

func (p *Plugin) httpChannelCreateSubscription(w http.ResponseWriter, r *http.Request, mattermostUserID string) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	if len(subscription.ChannelID) != 26 ||
		len(subscription.ID) != 0 {
		return respondErr(w, http.StatusBadRequest,
			fmt.Errorf("channel subscription invalid"))
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelID, mattermostUserID)
	if appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("not a member of the channel specified"))
	}

	err = p.hasPermissionToManageSubscription(subscription.InstanceID, mattermostUserID, subscription.ChannelID)
	if err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	client, _, connection, err := p.getClient(subscription.InstanceID, types.ID(mattermostUserID))
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
	p.UpdateUserDefaults(types.ID(mattermostUserID), subscription.InstanceID, projectKey)

	code, err := respondJSON(w, &subscription)
	if err != nil {
		return code, err
	}

	_, appErr = p.API.CreatePost(&model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelID,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was added to this channel by %v", subscription.Name, connection.DisplayName),
	})
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post"))
	}

	return http.StatusOK, nil
}

func (p *Plugin) httpChannelEditSubscription(w http.ResponseWriter, r *http.Request, mattermostUserID string) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	if len(subscription.ChannelID) != 26 ||
		len(subscription.ID) != 26 {
		return respondErr(w, http.StatusBadRequest,
			fmt.Errorf("channel subscription invalid"))
	}

	err = p.hasPermissionToManageSubscription(subscription.InstanceID, mattermostUserID, subscription.ChannelID)
	if err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelID, mattermostUserID)
	if appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("not a member of the channel specified"))
	}

	client, _, connection, err := p.getClient(subscription.InstanceID, types.ID(mattermostUserID))
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
	p.UpdateUserDefaults(types.ID(mattermostUserID), subscription.InstanceID, projectKey)

	code, err := respondJSON(w, &subscription)
	if err != nil {
		return code, err
	}

	_, appErr = p.API.CreatePost(&model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelID,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was updated by %v", subscription.Name, connection.DisplayName),
	})
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post"))
	}

	return http.StatusOK, nil
}

func (p *Plugin) httpChannelDeleteSubscription(w http.ResponseWriter, r *http.Request, mattermostUserID string) (int, error) {
	subscriptionID := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(subscriptionID) != 26 {
		return respondErr(w, http.StatusBadRequest,
			errors.New("bad subscription id"))
	}

	instanceID := types.ID(r.FormValue("instance_id"))
	subscription, err := p.getChannelSubscription(instanceID, subscriptionID)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.Wrap(err, "bad subscription id"))
	}

	err = p.hasPermissionToManageSubscription(instanceID, mattermostUserID, subscription.ChannelID)
	if err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	_, appErr := p.API.GetChannelMember(subscription.ChannelID, mattermostUserID)
	if appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("not a member of the channel specified"))
	}

	err = p.removeChannelSubscription(instanceID, subscriptionID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.Wrap(err, "unable to remove channel subscription"))
	}

	code, err := respondJSON(w, map[string]interface{}{"status": "OK"})
	if err != nil {
		return code, err
	}

	connection, err := p.userStore.LoadConnection(instanceID, types.ID(mattermostUserID))
	if err != nil {
		return http.StatusInternalServerError, err
	}
	_, appErr = p.API.CreatePost(&model.Post{
		UserId:    p.getConfig().botUserID,
		ChannelId: subscription.ChannelID,
		Message:   fmt.Sprintf("Jira subscription, \"%v\", was removed from this channel by %v", subscription.Name, connection.DisplayName),
	})
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post"))
	}
	return http.StatusOK, nil
}

func (p *Plugin) httpChannelGetSubscriptions(w http.ResponseWriter, r *http.Request, mattermostUserID string) (int, error) {
	channelID := strings.TrimPrefix(r.URL.Path, routeAPISubscriptionsChannel+"/")
	if len(channelID) != 26 {
		return respondErr(w, http.StatusBadRequest,
			errors.New("bad channel id"))
	}
	instanceID := types.ID(r.FormValue("instance_id"))

	if _, appErr := p.API.GetChannelMember(channelID, mattermostUserID); appErr != nil {
		return respondErr(w, http.StatusForbidden,
			errors.New("not a member of the channel specified"))
	}

	if err := p.hasPermissionToManageSubscription(instanceID, mattermostUserID, channelID); err != nil {
		return respondErr(w, http.StatusForbidden,
			errors.Wrap(err, "you don't have permission to manage subscriptions"))
	}

	subscriptions, err := p.getSubscriptionsForChannel(instanceID, channelID)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.Wrap(err, "unable to get channel subscriptions"))
	}

	return respondJSON(w, subscriptions)
}

func (p *Plugin) httpChannelSubscriptions(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	switch r.Method {
	case http.MethodPost:
		return p.httpChannelCreateSubscription(w, r, mattermostUserID)
	case http.MethodDelete:
		return p.httpChannelDeleteSubscription(w, r, mattermostUserID)
	case http.MethodGet:
		return p.httpChannelGetSubscriptions(w, r, mattermostUserID)
	case http.MethodPut:
		return p.httpChannelEditSubscription(w, r, mattermostUserID)
	default:
		return respondErr(w, http.StatusMethodNotAllowed, fmt.Errorf("Request: "+r.Method+" is not allowed."))
	}
}
