// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"github.com/pkg/errors"

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
			if errors.Is(err, errWebhookeventUnsupported) {
				ww.p.debugf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
			} else {
				ww.p.errorf("WebhookWorker id: %d, error processing, err: %v", ww.id, err)
			}
		}
	}
}

func (ww webhookWorker) process(msg *webhookMessage) (err error) {
	defer func() {
		if errors.Is(err, ErrWebhookIgnored) {
			// ignore ErrWebhookIgnored - from here up it's a success
			err = nil
		}
	}()

	wh, err := ParseWebhook(msg.Data)
	if err != nil {
		return err
	}

	v := wh.(*webhook)

	if isStandaloneCommentEvent(v.JiraWebhook) {
		instance, loadErr := ww.p.instanceStore.LoadInstance(msg.InstanceID)
		if loadErr != nil {
			return loadErr
		}
		if !instance.Common().IsCloudInstance() {
			return ErrWebhookIgnored
		}
	}

	if err = v.JiraWebhook.expandIssue(ww.p, msg.InstanceID); err != nil {
		return err
	}

	ww.p.checkIssueWatchers(v, msg.InstanceID)
	ww.p.applyReporterNotification(v, msg.InstanceID, v.Issue.Fields.Reporter)

	if _, _, err = wh.PostNotifications(ww.p, msg.InstanceID); err != nil {
		ww.p.errorf("WebhookWorker id: %d, error posting notifications, err: %v", ww.id, err)
	}

	channelsSubscribed, err := ww.p.getChannelsSubscribed(v, msg.InstanceID)
	if err != nil {
		return err
	}

	botUserID := ww.p.getUserID()
	for _, channelSubscribed := range channelsSubscribed {
		channel, err := ww.p.client.Channel.Get(channelSubscribed.ChannelID)
		if err != nil {
			ww.p.client.Log.Warn("Error occurred while getting the channel details while posting the webhook event", "ChannelID", channelSubscribed.ChannelID, "Error", err.Error())
			return err
		}

		if channel.DeleteAt > 0 {
			continue
		}

		// DM/GM subscriptions are legacy data; only keep posting to them while at least
		// one member is still connected to this instance.
		hasConnectedMember, err := ww.p.channelHasConnectedMember(msg.InstanceID, channel)
		if err != nil {
			ww.p.client.Log.Warn("Failed to check for connected members before posting subscription notification; skipping post",
				"ChannelID", channelSubscribed.ChannelID, "InstanceID", string(msg.InstanceID), "Error", err.Error())
			continue
		}
		if !hasConnectedMember {
			ww.p.client.Log.Info("Skipping subscription post to DM/GM channel with no connected members; removing orphaned subscriptions",
				"ChannelID", channelSubscribed.ChannelID, "InstanceID", string(msg.InstanceID))
			if removeErr := ww.p.removeSubscriptionsForChannel(msg.InstanceID, channelSubscribed.ChannelID); removeErr != nil {
				ww.p.client.Log.Warn("Failed to remove orphaned DM/GM subscriptions",
					"ChannelID", channelSubscribed.ChannelID, "InstanceID", string(msg.InstanceID), "Error", removeErr.Error())
			}
			continue
		}

		if _, _, err1 := wh.PostToChannel(ww.p, msg.InstanceID, channelSubscribed.ChannelID, botUserID, channelSubscribed.Name); err1 != nil {
			ww.p.errorf("WebhookWorker id: %d, error posting to channel, err: %v", ww.id, err1)
		}
	}

	return nil
}
