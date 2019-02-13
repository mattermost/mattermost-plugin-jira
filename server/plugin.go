// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"encoding/json"
	"io/ioutil"
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
	if config.Secret == "" || config.UserName == "" {
		http.Error(w, p.debug("JIRA plugin not configured correctly; must provide Secret and UserName"), http.StatusForbidden)
		return
	}

	if r.URL.Path != "/webhook" {
		http.Error(w, p.debug("Request URL: unsupported path: "+r.URL.Path+", must be /webhook"), http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, p.debug("Request: "+r.Method+" is not allowed, must be POST"), http.StatusMethodNotAllowed)
		return
	}

	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(config.Secret)) != 1 {
		http.Error(w, p.debug("Request URL: secret did not match"), http.StatusForbidden)
		return
	}

	teamName := r.URL.Query().Get("team")
	if teamName == "" {
		http.Error(w, p.debug("Request URL: team is empty"), http.StatusBadRequest)
		return
	}
	channelID := r.URL.Query().Get("channel")
	if channelID == "" {
		http.Error(w, p.debug("Request URL: channel is empty"), http.StatusBadRequest)
		return
	}

	user, appErr := p.API.GetUserByUsername(config.UserName)
	if appErr != nil {
		http.Error(w, p.debug(appErr.Message), appErr.StatusCode)
		return
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelID, false)
	if appErr != nil {
		http.Error(w, p.debug(appErr.Message), appErr.StatusCode)
		return
	}

	bb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, p.debug(err.Error()), http.StatusBadRequest)
		return
	}

	var webhook Webhook
	err = json.Unmarshal(bb, &webhook)
	if err != nil {
		http.Error(w, p.debug(err.Error()), http.StatusBadRequest)
		return
	}
	if webhook.WebhookEvent == "" {
		http.Error(w, p.debug("No webhook event"), http.StatusBadRequest)
		return
	}
	webhook.RawJSON = string(bb)

	message := webhook.Markdown()
	if message == "" {
		return
	}
	post := &model.Post{
		ChannelId: channel.Id,
		UserId:    user.Id,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
		},
		Message: message,
	}

	_, appErr = p.API.CreatePost(post)
	if appErr != nil {
		http.Error(w, p.debug(appErr.Message), appErr.StatusCode)
		return
	}
}

func (p *Plugin) debug(errorMessage string) string {
	p.API.LogDebug("Failed to serve HTTP request", "Error message", errorMessage)
	return errorMessage
}
