package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Configuration struct {
	Enabled  bool
	Secret   string
	UserName string
}

type Plugin struct {
	api           plugin.API
	configuration atomic.Value
}

func (p *Plugin) OnActivate(api plugin.API) error {
	p.api = api
	return p.OnConfigurationChange()
}

func (p *Plugin) config() *Configuration {
	return p.configuration.Load().(*Configuration)
}

func (p *Plugin) OnConfigurationChange() error {
	var configuration Configuration
	if err := p.api.LoadPluginConfiguration(&configuration); err != nil {
		return err
	}
	p.configuration.Store(&configuration)
	return nil
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	config := p.config()
	if !config.Enabled || config.Secret == "" || config.UserName == "" {
		http.Error(w, "This plugin is not configured.", http.StatusForbidden)
		return
	} else if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("secret")), []byte(config.Secret)) != 1 {
		http.Error(w, "You must provide the configured secret.", http.StatusForbidden)
		return
	} else if r.URL.Path != "/webhook" {
		http.NotFound(w, r)
		return
	} else if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
	} else if user, err := p.api.GetUserByUsername(config.UserName); err != nil {
		http.Error(w, err.Message, err.StatusCode)
	} else if team, err := p.api.GetTeamByName(r.URL.Query().Get("team")); err != nil {
		http.Error(w, err.Message, err.StatusCode)
	} else if channel, err := p.api.GetChannelByName(team.Id, r.URL.Query().Get("channel")); err != nil {
		http.Error(w, err.Message, err.StatusCode)
	} else if _, err := p.api.CreatePost(&model.Post{
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
