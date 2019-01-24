// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("HTTP request", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())

	config := p.getConfiguration()
	if !config.Enabled || config.Secret == "" || config.UserName == "" {
		errorMessage := "This plugin is not configured"
		p.postHTTPDebugMessage(errorMessage)
		http.Error(w, errorMessage, http.StatusForbidden)
		return
	}

	if r.URL.Path != "/webhook" {
		p.postHTTPDebugMessage("Invalid URL path")
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		errorMessage := "Method not allowed"
		p.postHTTPDebugMessage(errorMessage)
		http.Error(w, errorMessage, http.StatusMethodNotAllowed)
		return
	}

	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(config.Secret)) != 1 {
		errorMessage := "You must provide the configured secret"
		p.postHTTPDebugMessage(errorMessage)
		http.Error(w, errorMessage, http.StatusForbidden)
		return
	}

	teamName := r.URL.Query().Get("team")
	if teamName == "" {
		errorMessage := "You must provide a teamName"
		p.postHTTPDebugMessage(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	channelID := r.URL.Query().Get("channel")
	if channelID == "" {
		errorMessage := "You must provide a channelID"
		p.postHTTPDebugMessage(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	var webhook *Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil || webhook == nil {
		p.postHTTPDebugMessage(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := webhook.IsValid(); err != nil {
		p.postHTTPDebugMessage(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	attachment, err := webhook.SlackAttachment()
	if err != nil {
		p.postHTTPDebugMessage(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if attachment == nil {
		errorMessage := "Failed to create post"
		p.postHTTPDebugMessage(errorMessage)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}

	user, appErr := p.API.GetUserByUsername(config.UserName)
	if appErr != nil {
		p.postHTTPDebugMessage(appErr.Message)
		http.Error(w, appErr.Message, appErr.StatusCode)
		return
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelID, false)
	if appErr != nil {
		p.postHTTPDebugMessage(appErr.Message)
		http.Error(w, appErr.Message, appErr.StatusCode)
		return
	}

	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    user.Id,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
		},
	}
	model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})
	if _, appErr := p.API.CreatePost(post); appErr != nil {
		p.postHTTPDebugMessage(appErr.Message)
		http.Error(w, appErr.Message, appErr.StatusCode)
		return
	}
}

func (p *Plugin) postHTTPDebugMessage(errorMessage string) {
	p.API.LogDebug("Failed to serve HTTP request", "Error message", errorMessage)
}
