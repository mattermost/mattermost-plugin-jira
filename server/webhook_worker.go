// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type webhookWorker struct {
	id        int
	p         *Plugin
	workQueue <-chan *webhookMessage
}

type webhookMessage struct {
	InstanceID types.ID
	Data       []byte
}

func (ww webhookWorker) work() {
	for msg := range ww.workQueue {
		err := ww.process(msg)
		if err != nil {
			ww.p.errorf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
		}
	}
}

func (ww webhookWorker) process(msg *webhookMessage) (err error) {
	conf := ww.p.getConfig()
	start := time.Now()
	defer func() {
		isError, isIgnored := false, false
		switch err {
		case nil:
			break
		case ErrWebhookIgnored:
			// ignore ErrWebhookIgnored - from here up it's a success
			isIgnored = true
			err = nil
		default:
			// TODO save the payload here
			isError = true
		}
		if conf.stats != nil {
			path := instancePath("jira/subscribe/processing", msg.InstanceID)
			conf.stats.EnsureEndpoint(path).Record(utils.ByteSize(len(msg.Data)), 0, time.Since(start), isError, isIgnored)
		}
	}()

	wh, err := ParseWebhook(msg.Data)
	if err != nil {
		return err
	}
	v := wh.(*webhook)
	if err = v.JiraWebhook.expandIssue(ww.p, msg.InstanceID); err != nil {
		return err
	}
	err = wh.(*webhook).FetchReporterNotification(msg.Data, v.Issue.Fields.Reporter)
	if _, _, err = wh.PostNotifications(ww.p, msg.InstanceID); err != nil {
		ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
	}

	channelsSubscribed, err := ww.p.getChannelsSubscribed(v, msg.InstanceID)
	if err != nil {
		return err
	}

	botUserID := ww.p.getUserID()
	for _, channelSubscribed := range channelsSubscribed {
		if _, _, err1 := wh.PostToChannel(ww.p, msg.InstanceID, channelSubscribed.ChannelID, botUserID, channelSubscribed.Name); err1 != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err1)
		}
	}

	return nil
}

func (wh *webhook) FetchReporterNotification(bb []byte, reporter *jira.User) error {
	if wh.eventTypes.ContainsAny("event_created_comment") || wh.eventTypes.ContainsAny("event_updated_comment") {
		jwhook := &JiraWebhook{}
		err := json.Unmarshal(bb, &jwhook)
		if err != nil {
			return err
		}
		if jwhook.Issue.ID == "" {
			return ErrWebhookIgnored
		}

		commentAuthor := mdUser(&jwhook.Comment.UpdateAuthor)

		whook := &webhook{
			JiraWebhook: jwhook,
			eventTypes:  NewStringSet(eventCreatedComment),
			headline:    fmt.Sprintf("%s **commented** on %s", commentAuthor, jwhook.mdKeySummaryLink()),
			text:        truncate(quoteIssueComment(jwhook.Comment.Body), 3000),
		}
		jwh := whook.JiraWebhook

		commentAuthor = mdUser(&jwh.Comment.UpdateAuthor)

		commentMessage := fmt.Sprintf("%s **commented** on %s:\n>%s", commentAuthor, jwh.mdKeySummaryLink(), jwh.Comment.Body)
		if reporter == nil ||
			(reporter.Name != "" && reporter.Name == jwh.User.Name) ||
			(reporter.AccountID != "" && reporter.AccountID == jwh.Comment.UpdateAuthor.AccountID) {
			return nil
		}
		wh.notifications = append(wh.notifications, webhookUserNotification{
			jiraUsername:  reporter.Name,
			jiraAccountID: reporter.AccountID,
			message:       commentMessage,
			postType:      PostTypeComment,
			commentSelf:   jwh.Comment.Self,
			recipientType: recipientTypeReporter,
		})
	}
	return nil
}
