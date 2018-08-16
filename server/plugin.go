package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	KEY_SECURITY_CONTEXT = "security_context"
)

type Plugin struct {
	plugin.MattermostPlugin

	Enabled  bool
	Secret   string
	UserName string

	securityContext *SecurityContext
}

type SecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	PublicKey      string `json:"publicKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if !p.Enabled || p.Secret == "" || p.UserName == "" {
		http.Error(w, "This plugin is not configured.", http.StatusForbidden)
		return
	}

	switch r.URL.Path {
	case "/webhook":
		p.serveWebhook(w, r)
		return
	case "/atlassian-connect.json":
		p.serveAtlassianConnect(w, r)
		return
	case "/installed":
		p.serveInstalled(w, r)
		return
	case "/issue_event":
		p.serveIssueEvent(w, r)
		return
	}

	http.NotFound(w, r)
}

func (p *Plugin) serveAtlassianConnect(w http.ResponseWriter, r *http.Request) {
	config := p.API.GetConfig()
	baseURL := *config.ServiceSettings.SiteURL + "/" + path.Join("plugins", PluginId)

	lp := filepath.Join(*config.PluginSettings.Directory, PluginId, "server", "dist", "templates", "atlassian-connect.json")
	vals := map[string]string{
		"BaseURL": baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl.ExecuteTemplate(w, "config", vals)
}

func (p *Plugin) serveInstalled(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("Can't read request:%v\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	var sc SecurityContext
	json.Unmarshal(body, &sc)

	p.securityContext = &sc

	p.API.KVSet(KEY_SECURITY_CONTEXT, body)

	json.NewEncoder(w).Encode([]string{"OK"})
}

func (p *Plugin) serveIssueEvent(w http.ResponseWriter, r *http.Request) {
}

func (p *Plugin) serveWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
