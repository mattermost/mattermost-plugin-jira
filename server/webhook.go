// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	webhookDedupTTL         = 30 * time.Second
	notificationDedupKeyFmt = "notif_dedup_%s"
	channelPostDedupKeyFmt  = "chan_dedup_%s"
)

const (
	Nobody         = "_nobody_"
	commentDeleted = "comment_deleted"
	commentUpdated = "comment_updated"
	commentCreated = "comment_created"
	issueCreated   = "jira:issue_created"

	worklogUpdated = "jira:worklog_updated"

	ticketRootPostIDKey = "ticket_post_id_%s_channel_id_%s"
)

type Webhook interface {
	Events() StringSet
	PostToChannel(p *Plugin, instanceID types.ID, channelID, fromUserID, subscriptionName string) (*model.Post, int, error)
	PostNotifications(p *Plugin, instanceID types.ID) ([]*model.Post, int, error)
}

type webhookField struct {
	name string
	id   string
	from string
	to   string
}

type webhook struct {
	*JiraWebhook
	eventTypes    StringSet
	headline      string
	text          string
	fields        []*model.SlackAttachmentField
	notifications []webhookUserNotification
	fieldInfo     webhookField
}

type webhookUserNotification struct {
	jiraUsername     string
	jiraAccountID    string
	message          string
	postType         string
	commentSelf      string
	notificationType string
}

func (wh *webhook) Events() StringSet {
	return wh.eventTypes
}

func (wh webhook) PostToChannel(p *Plugin, instanceID types.ID, channelID, fromUserID, subscriptionName string) (*model.Post, int, error) {
	pluginConfig := p.getConfig()

	if wh.headline == "" {
		return nil, http.StatusBadRequest, errors.Errorf("unsupported webhook")
	}

	// Keep the dedup identity independent of the subscription name so overlapping
	// subscriptions on the same channel collapse into one post.
	headline := wh.headline
	if pluginConfig.DisplaySubscriptionNameInNotifications && subscriptionName != "" {
		headline = fmt.Sprintf("%s\nSubscription: **%s**", headline, subscriptionName)
	}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    fromUserID,
	}

	key := fmt.Sprintf(ticketRootPostIDKey, wh.Issue.ID, channelID)
	var rootID string
	rootPostExists := false

	_, hasCreatedComment := wh.eventTypes[eventCreatedComment]
	_, hasDeletedComment := wh.eventTypes[eventDeletedComment]
	_, hasUpdatedComment := wh.eventTypes[eventUpdatedComment]

	if hasCreatedComment || hasDeletedComment || hasUpdatedComment {
		err := p.client.KV.Get(key, &rootID)
		if err != nil || rootID == "" {
			p.client.Log.Info("Post ID not found in KV store, creating a new post for Jira subscription comment event", "TicketID", wh.Issue.ID)
		} else {
			rootPostExists = true
			post.RootId = rootID
		}
	}

	text := ""
	if wh.text != "" && !pluginConfig.HideDecriptionComment {
		var client Client
		if p.userStore != nil {
			client, _, _ = wh.fetchConnectedUser(p, instanceID)
		}
		text = p.replaceJiraAccountIds(instanceID, wh.text, client)
	}

	if text != "" || len(wh.fields) != 0 {
		model.ParseSlackAttachment(post, []*model.SlackAttachment{
			{
				// TODO is this supposed to be themed?
				Color:    "#95b7d0",
				Fallback: headline,
				Pretext:  headline,
				Text:     text,
				Fields:   wh.fields,
			},
		})
	} else {
		post.Message = headline
	}

	// Atomically claim the dedup key before posting so concurrent webhook
	// deliveries can't both pass the check and post duplicates. SetAtomic(nil)
	// only writes when the key does not already exist.
	dedupKey := channelPostDedupKey(instanceID, &wh, channelID)
	claimed, kvErr := p.client.KV.Set(dedupKey, true, pluginapi.SetExpiry(webhookDedupTTL), pluginapi.SetAtomic(nil))
	switch {
	case kvErr != nil:
		// Fail open: post rather than dropping the event.
		p.client.Log.Warn("PostToChannel: failed to claim dedup key, posting anyway", "key", dedupKey, "error", kvErr.Error())
	case !claimed:
		// Another delivery already claimed this post.
		p.client.Log.Debug("PostToChannel: another delivery already claimed this post, skipping", "key", dedupKey)
		return nil, http.StatusOK, nil
	}

	if err := p.client.Post.CreatePost(post); err != nil {
		if claimed {
			p.releaseDedupKey("PostToChannel", dedupKey)
		}
		return nil, http.StatusInternalServerError, err
	}

	commentEvent := commentEvents.ContainsAny(wh.Events().ToSlice()...)
	issueCreated := wh.eventTypes[eventCreated]
	shouldStorePostID := commentEvent || issueCreated

	if shouldStorePostID && !rootPostExists {
		commentPostReplyDuration, err := strconv.Atoi(pluginConfig.ThreadedJiraCommentSubscriptionDuration)
		if err != nil {
			p.client.Log.Error("Error converting comment post reply duration to integer, future comments may not thread correctly", "TicketID", wh.Issue.ID, "PostID", post.Id, "Error", err.Error())
		} else {
			expiry := pluginapi.SetExpiry(time.Duration(commentPostReplyDuration) * 24 * time.Hour)
			if _, err := p.client.KV.Set(key, post.Id, expiry); err != nil {
				p.client.Log.Error("Failed to store post ID, future comments may not thread correctly", "TicketID", wh.Issue.ID, "PostID", post.Id, "error", err.Error())
			}
		}
	}

	return post, http.StatusOK, nil
}

func (wh *webhook) PostNotifications(p *Plugin, instanceID types.ID) ([]*model.Post, int, error) {
	if len(wh.notifications) == 0 {
		return nil, http.StatusOK, nil
	}

	// We will only send webhook events if we have a connected instance.
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		// This isn't an internal server error. There's just no instance installed.
		return nil, http.StatusOK, nil
	}

	posts := []*model.Post{}
	var mapForNotification = make(map[types.ID]int)
	for _, notification := range wh.notifications {
		var mattermostUserID types.ID
		var err error

		// prefer accountId to username when looking up UserIds
		if notification.jiraAccountID != "" {
			mattermostUserID, err = p.userStore.LoadMattermostUserID(instance.GetID(), notification.jiraAccountID)
		} else {
			mattermostUserID, err = p.userStore.LoadMattermostUserID(instance.GetID(), notification.jiraUsername)
		}
		if err != nil {
			continue
		}

		// Check if the user has permissions.
		c, err2 := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
		if err2 != nil {
			// Not connected to Jira, so can't check permissions
			continue
		}
		client, err2 := instance.GetClient(c)
		if err2 != nil {
			p.errorf("PostNotifications: error while getting jiraClient, err: %v", err2)
			continue
		}
		// If this is a comment-related webhook, we need to check if they have permissions to read that.
		// Otherwise, check if they can view the issue.

		if !c.Settings.ShouldReceiveNotification(notification.notificationType) {
			continue
		}

		if _, ok := mapForNotification[mattermostUserID]; ok {
			continue
		}
		mapForNotification[mattermostUserID] = 1

		isCommentEvent := wh.Events().Intersection(commentEvents).Len() > 0
		if isCommentEvent {
			if instance.Common().IsCloudInstance() {
				err = client.RESTGet(fmt.Sprintf("/2/issue/%s/comment/%s", wh.Issue.ID, wh.Comment.ID), nil, &struct{}{})
			} else {
				err = client.RESTGet(notification.commentSelf, nil, &struct{}{})
			}
		} else {
			_, err = client.GetIssue(wh.Issue.ID, nil)
		}
		if err != nil {
			p.errorf("PostNotifications: failed to get self: %v", err)
			continue
		}

		notification.message = p.replaceJiraAccountIds(instance.GetID(), notification.message, client)

		// Atomically claim the dedup key before posting so concurrent webhook
		// deliveries can't both pass the check and send duplicates. SetAtomic(nil)
		// only writes when the key does not already exist.
		dedupKey := notificationDedupKey(instance.GetID(), wh, mattermostUserID, notification.message)
		claimed, kvErr := p.client.KV.Set(dedupKey, true, pluginapi.SetExpiry(webhookDedupTTL), pluginapi.SetAtomic(nil))
		switch {
		case kvErr != nil:
			// Fail open: send the notification rather than dropping it.
			p.client.Log.Warn("PostNotifications: failed to claim dedup key, sending notification anyway", "key", dedupKey, "error", kvErr.Error())
		case !claimed:
			// Another delivery already claimed this notification.
			continue
		}

		post, err := p.CreateBotDMPost(instance.GetID(), mattermostUserID, notification.message, notification.postType)
		if err != nil {
			p.errorf("PostNotifications: failed to create notification post, err: %v", err)
			if claimed {
				p.releaseDedupKey("PostNotifications", dedupKey)
			}
			continue
		}
		posts = append(posts, post)
	}
	return posts, http.StatusOK, nil
}

func newWebhook(jwh *JiraWebhook, eventType string, format string, args ...interface{}) *webhook {
	return &webhook{
		JiraWebhook: jwh,
		eventTypes:  NewStringSet(eventType),
		headline:    jwh.mdUser() + " " + fmt.Sprintf(format, args...) + " " + jwh.mdKeySummaryLink(),
	}
}

// releaseDedupKey drops a dedup claim after the post it guarded failed, so a
// later delivery of the same event can retry instead of being skipped until the
// claim expires. Only safe to call when this delivery won the claim.
func (p *Plugin) releaseDedupKey(caller, dedupKey string) {
	if err := p.client.KV.Delete(dedupKey); err != nil {
		p.client.Log.Warn(caller+": failed to release dedup key after a failed post; duplicates of this event will be skipped until the claim expires",
			"key", dedupKey, "error", err.Error())
	}
}

// writeDedupToken writes s into h prefixed with its length, so concatenated
// tokens can't collide due to separator characters appearing inside values.
func writeDedupToken(h hash.Hash, s string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(s)))
	h.Write(length[:]) //nolint:errcheck // hash.Hash.Write never returns an error
	h.Write([]byte(s)) //nolint:errcheck // hash.Hash.Write never returns an error
}

func notificationDedupKey(instanceID types.ID, wh *webhook, recipientID types.ID, message string) string {
	h := sha256.New()
	for _, token := range []string{string(instanceID), wh.Issue.Key, string(recipientID), message} {
		writeDedupToken(h, token)
	}
	return fmt.Sprintf(notificationDedupKeyFmt, hex.EncodeToString(h.Sum(nil)))
}

func channelPostDedupKey(instanceID types.ID, wh *webhook, channelID string) string {
	h := sha256.New()
	for _, token := range []string{string(instanceID), wh.Issue.Key, channelID, wh.headline, wh.text} {
		writeDedupToken(h, token)
	}
	for _, f := range wh.fields {
		writeDedupToken(h, f.Title)
		writeDedupToken(h, fmt.Sprintf("%v", f.Value))
		writeDedupToken(h, fmt.Sprintf("%t", bool(f.Short)))
	}
	return fmt.Sprintf(channelPostDedupKeyFmt, hex.EncodeToString(h.Sum(nil)))
}

func (p *Plugin) GetWebhookURL(jiraURL string, teamID, channelID string) (subURL, legacyURL string, err error) {
	cf := p.getConfig()

	instanceID, err := p.ResolveWebhookInstanceURL(jiraURL)
	if err != nil {
		return "", "", err
	}

	team, err := p.client.Team.Get(teamID)
	if err != nil {
		return "", "", err
	}

	channel, err := p.client.Channel.Get(channelID)
	if err != nil {
		return "", "", err
	}

	v := url.Values{}
	v.Add("secret", cf.Secret)
	subURL = p.GetPluginURL() + instancePath(makeAPIRoute(routeAPISubscribeWebhook), instanceID) + "?" + v.Encode()

	// For the legacy URL, add team and channel. Secret is already in the map.
	v.Add("team", team.Name)
	v.Add("channel", channel.Name)
	legacyURL = p.GetPluginURL() + instancePath(routeIncomingWebhook, instanceID) + "?" + v.Encode()

	return subURL, legacyURL, nil
}

func (p *Plugin) getSubscriptionsWebhookURL(instanceID types.ID) string {
	cf := p.getConfig()
	v := url.Values{}
	v.Add("secret", cf.Secret)
	return p.GetPluginURL() + instancePath(makeAPIRoute(routeAPISubscribeWebhook), instanceID) + "?" + v.Encode()
}
