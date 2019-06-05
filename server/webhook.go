// See License for license information.
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.

package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Webhook interface {
	EventMask() uint64
	PostToChannel(api plugin.API, channelId, fromUserId string) (*model.Post, int, error)
	PostNotifications(Config, plugin.API, UserStore, Instance) ([]*model.Post, int, error)
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

func (wh webhook) PostToChannel(api plugin.API, channelId, fromUserId string) (*model.Post, int, error) {
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

	_, appErr := api.CreatePost(post)
	if appErr != nil {
		return nil, appErr.StatusCode, appErr
	}

	return post, http.StatusOK, nil
}

func (wh *webhook) PostNotifications(conf Config, api plugin.API, userStore UserStore,
	instance Instance) ([]*model.Post, int, error) {

	posts := []*model.Post{}
	if len(wh.notifications) == 0 {
		return nil, http.StatusOK, nil
	}
	for _, notification := range wh.notifications {
		mattermostUserId, err := userStore.LoadMattermostUserId(
			instance, notification.jiraUsername)
		if err != nil {
			return nil, http.StatusOK, nil
		}

		post, err := CreateBotDMPost(conf, api, userStore, instance, mattermostUserId,
			notification.message, notification.postType)
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

func GetWebhookURL(conf Config, api plugin.API, teamId, channelId string) (string, error) {
	team, appErr := api.GetTeam(teamId)
	if appErr != nil {
		return "", appErr
	}

	channel, appErr := api.GetChannel(channelId)
	if appErr != nil {
		return "", appErr
	}

	v := url.Values{}
	secret, _ := url.QueryUnescape(conf.Secret)
	v.Add("secret", secret)
	v.Add("team", team.Name)
	v.Add("channel", channel.Name)
	return conf.PluginURL + "/" + routeIncomingWebhook + "?" + v.Encode(), nil
}
