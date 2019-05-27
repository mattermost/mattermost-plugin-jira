// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

type Webhook interface {
	EventMask() uint64
	PostToChannel(p *Plugin, channelId, fromUserId string) (*model.Post, int, error)
	PostNotifications(p *Plugin, ji Instance) ([]*model.Post, int, error)
}

type webhook struct {
	*JiraWebhook
	eventMask     uint64
	headline      string
	text          string
	fields        []*model.SlackAttachmentField
	notifications []webhookNotification
}

type webhookNotification struct {
	jiraUsername string
	message      string
	postType     string
}

func (wh *webhook) EventMask() uint64 {
	return wh.eventMask
}

func (wh webhook) PostToChannel(p *Plugin, channelId, fromUserId string) (*model.Post, int, error) {
	if wh.headline == "" {
		return nil, http.StatusBadRequest, errors.Errorf("unsupported webhook")
	}

	post := &model.Post{
		ChannelId: channelId,
		UserId:    fromUserId,
		// Props: map[string]interface{}{
		// 	"from_webhook":  "true",
		// 	"use_user_icon": "true",
		// },
	}
	if wh.text != "" || len(wh.fields) != 0 {
		model.ParseSlackAttachment(post, []*model.SlackAttachment{
			{
				// TODO is this supposed to be themed?
				Color:    "#95b7d0",
				Fallback: wh.headline,
				Pretext:  wh.headline,
				Text:     wh.text,
				Fields:   wh.fields,
			},
		})
	} else {
		post.Message = wh.headline
	}

	_, appErr := p.API.CreatePost(post)
	if appErr != nil {
		return nil, appErr.StatusCode, appErr
	}

	return post, http.StatusOK, nil
}

func (wh *webhook) PostNotifications(p *Plugin, ji Instance) ([]*model.Post, int, error) {
	posts := []*model.Post{}
	if len(wh.notifications) == 0 {
		return nil, http.StatusOK, nil
	}
	for _, notification := range wh.notifications {
		mattermostUserId, err := p.userStore.LoadMattermostUserId(
			ji, notification.jiraUsername)
		if err != nil {
			return nil, http.StatusOK, nil
		}

		post, err := ji.GetPlugin().CreateBotDMPost(ji, mattermostUserId, notification.message, notification.postType)
		if err != nil {
			return nil, http.StatusInternalServerError, errors.WithMessage(err, "failed to create notification post")
		}
		posts = append(posts, post)
	}
	return posts, http.StatusOK, nil
}

func newWebhook(jwh *JiraWebhook, eventMask uint64, format string, args ...interface{}) *webhook {
	return &webhook{
		JiraWebhook: jwh,
		eventMask:   eventMask,
		headline:    jwh.mdUser() + " " + fmt.Sprintf(format, args...) + " " + jwh.mdKeyLink(),
	}
}

func (p *Plugin) GetWebhookURL(teamId, channelId string) (string, error) {
	cf := p.getConfig()

	team, appErr := p.API.GetTeam(teamId)
	if appErr != nil {
		return "", appErr
	}

	channel, appErr := p.API.GetChannel(channelId)
	if appErr != nil {
		return "", appErr
	}

	v := url.Values{}
	secret, _ := url.QueryUnescape(cf.Secret)
	v.Add("secret", secret)
	v.Add("team", team.Name)
	v.Add("channel", channel.Name)
	return p.GetPluginURL() + "/" + routeIncomingWebhook + "?" + v.Encode(), nil
}
