// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
)

type WebhookUser struct {
	Self         string
	Name         string
	Key          string
	EmailAddress string
	AvatarURLs   map[string]string
	DisplayName  string
	Active       bool
	TimeZone     string
}

type Webhook struct {
	WebhookEvent string
	Issue        struct {
		Self   string
		Key    string
		Fields struct {
			Assignee    *WebhookUser
			Reporter    *WebhookUser
			Summary     string
			Description string
			Priority    *struct {
				Id      string
				Name    string
				IconURL string
			}
			IssueType struct {
				Name    string
				IconURL string
			}
			Resolution *struct {
				Id string
			}
			Status struct {
				Id string
			}
			Labels []string
		}
	}
	User    WebhookUser
	Comment struct {
		Body         string
		UpdateAuthor WebhookUser
	}
	ChangeLog struct {
		Items []struct {
			From       string
			FromString string
			To         string
			ToString   string
			Field      string
		}
	}
	IssueEventTypeName string `json:"issue_event_type_name"`
}

func (p *Plugin) handleHTTPWebhook(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}
	// TODO redo with JWT
	// if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(config.Secret)) != 1 {
	// 	return http.StatusForbidden,
	// 		fmt.Errorf("Request URL: secret did not match")
	// }

	teamName := r.URL.Query().Get("team")
	if teamName == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: team is empty")
	}
	channelID := r.URL.Query().Get("channel")
	if channelID == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Request URL: channel is empty")
	}

	config := p.getConfiguration()
	user, appErr := p.API.GetUserByUsername(config.UserName)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelID, false)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	initPost, err := AsSlackAttachment(r.Body)
	if err != nil {
		return http.StatusBadRequest, err
	}

	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    user.Id,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
		},
	}
	initPost(post)

	_, appErr = p.API.CreatePost(post)
	if appErr != nil {
		return appErr.StatusCode, fmt.Errorf(appErr.Message)
	}

	return http.StatusOK, nil
}
