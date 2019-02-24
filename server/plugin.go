// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"fmt"
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

	err, status := p.handleHTTPRequest(c, w, r)
	if err != nil {
		p.API.LogDebug("Failed to serve HTTP request", "Error message", err.Error())
		http.Error(w, err.Error(), status)
	}
}

func (p *Plugin) handleHTTPRequest(c *plugin.Context, w http.ResponseWriter, r *http.Request) (error, int) {
	config := p.getConfiguration()
	if config.Secret == "" || config.UserName == "" {
		return fmt.Errorf("JIRA plugin not configured correctly; must provide Secret and UserName"), http.StatusForbidden
	}

	if r.URL.Path != "/webhook" {
		return fmt.Errorf("Request URL: unsupported path: " + r.URL.Path + ", must be /webhook"), http.StatusNotFound
	}
	if r.Method != http.MethodPost {
		return fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST"), http.StatusMethodNotAllowed
	}

	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(config.Secret)) != 1 {
		return fmt.Errorf("Request URL: secret did not match"), http.StatusForbidden
	}

	teamName := r.URL.Query().Get("team")
	if teamName == "" {
		return fmt.Errorf("Request URL: team is empty"), http.StatusBadRequest
	}
	channelID := r.URL.Query().Get("channel")
	if channelID == "" {
		return fmt.Errorf("Request URL: channel is empty"), http.StatusBadRequest
	}

	user, appErr := p.API.GetUserByUsername(config.UserName)
	if appErr != nil {
		return fmt.Errorf(appErr.Message), appErr.StatusCode
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelID, false)
	if appErr != nil {
		return fmt.Errorf(appErr.Message), appErr.StatusCode
	}

	initPost, err := AsSlackAttachment(r.Body)
	if err != nil {
		return err, http.StatusBadRequest
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
		return fmt.Errorf(appErr.Message), appErr.StatusCode
	}

	return nil, http.StatusOK
}
