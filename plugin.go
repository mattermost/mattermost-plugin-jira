package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	Enabled  bool
	Secret   string
	UserName string
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if !p.Enabled || p.Secret == "" || p.UserName == "" {
		http.Error(w, "This plugin is not configured.", http.StatusForbidden)
		return
	} else if r.URL.Path != "/webhook" {
		http.NotFound(w, r)
		return
	} else if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	} else if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(p.Secret)) != 1 {
		http.Error(w, "You must provide the configured secret.", http.StatusForbidden)
		return
	}

	var webhook Webhook

	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if attachment, err := webhook.SlackAttachment(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else if attachment == nil {
		return
	} else if r.URL.Query().Get("channel") == "" {
		http.Error(w, "You must provide a channel.", http.StatusBadRequest)
	} else if user, err := p.API.GetUserByUsername(p.UserName); err != nil {
		http.Error(w, err.Message, err.StatusCode)
	} else if channel, err := p.API.GetChannelByNameForTeamName(r.URL.Query().Get("team"), r.URL.Query().Get("channel")); err != nil {
		http.Error(w, err.Message, err.StatusCode)
	} else if _, err := p.API.CreatePost(&model.Post{
		ChannelId: channel.Id,
		Type:      model.POST_SLACK_ATTACHMENT,
		UserId:    user.Id,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   []*model.SlackAttachment{attachment},
		},
	}); err != nil {
		http.Error(w, err.Message, err.StatusCode)
	}
}
