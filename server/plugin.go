package main

import (
	"context"
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

	jira "github.com/andygrunwald/go-jira"
	jwt "golang.org/x/oauth2/jira"
)

const (
	KEY_SECURITY_CONTEXT = "security_context"
	KEY_USER_INFO        = "user_info_"
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
	OAuthClientId  string `json:"oauthClientId"`
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if !p.Enabled || p.Secret == "" || p.UserName == "" {
		http.Error(w, "This plugin is not configured.", http.StatusForbidden)
		return
	}

	switch r.URL.Path {
	case "/test":
		p.serveTest(w, r)
		return
	case "/connect":
		p.serveUserConnectPage(w, r)
		return
	case "/connect/complete":
		p.serveUserConnectComplete(w, r)
		return
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

func (p *Plugin) serveUserConnectPage(w http.ResponseWriter, r *http.Request) {
	jiraURL := r.URL.Query().Get("xdm_e")

	config := p.API.GetConfig()
	completeURL := *config.ServiceSettings.SiteURL + "/" + path.Join("plugins", PluginId, "connect", "complete")

	html := `
	<!DOCTYPE html>
	<html>
		<head>
			<script src="%s/atlassian-connect/all.js"></script>
			<script>
				AP.getCurrentUser(function(user){
					console.log("user id:", user.atlassianAccountId);
					window.open("%s?account_id=" + user.atlassianAccountId);
				});
			</script>
		</head>
		<body>
			<p>From the Mattermost JIRA plugin.</p>
		</body>
	</html>
	`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(html, jiraURL, completeURL)))
}

type JiraUserInfo struct {
	AccountId string
}

func (p *Plugin) serveUserConnectComplete(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	info := &JiraUserInfo{AccountId: r.URL.Query().Get("account_id")}

	if info.AccountId == "" {
		http.Error(w, "Missing account_id", http.StatusBadRequest)
		return
	}

	b, _ := json.Marshal(info)

	p.API.KVSet(KEY_USER_INFO+userID, b)

	jiraClient, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		http.Error(w, "could not get jira client, err="+err.Error(), 500)
	}

	user, _, err := jiraClient.User.GetSelf()
	if err != nil {
		http.Error(w, "could not get the user, err="+err.Error(), 500)
	}

	userBytes, _ := json.Marshal(user)
	w.Header().Set("Content-Type", "application/json")
	w.Write(userBytes)
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

func (p *Plugin) getJiraUserInfo(userID string) (*JiraUserInfo, error) {
	b, _ := p.API.KVGet(KEY_USER_INFO + userID)
	if b == nil {
		return nil, fmt.Errorf("could not find jira user info")
	}

	info := JiraUserInfo{}
	err := json.Unmarshal(b, &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func (p *Plugin) serveTest(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	info, err := p.getJiraUserInfo(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	jiraClient, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		http.Error(w, "could not get jira client, err="+err.Error(), 500)
	}

	user, _, err := jiraClient.User.GetSelf()
	if err != nil {
		http.Error(w, "could not get the user, err="+err.Error(), 500)
	}

	userBytes, _ := json.Marshal(user)
	w.Header().Set("Content-Type", "application/json")
	w.Write(userBytes)
}

func (p *Plugin) loadSecurityContext() {
	b, _ := p.API.KVGet(KEY_SECURITY_CONTEXT)
	var sc SecurityContext
	json.Unmarshal(b, &sc)
	p.securityContext = &sc
}

func (p *Plugin) getJIRAClientForUser(jiraUser string) (*jira.Client, error) {
	if p.securityContext == nil {
		p.loadSecurityContext()
	}

	c := jwt.Config{
		BaseURL: p.securityContext.BaseURL,
		Subject: jiraUser,
	}

	c.Config.ClientID = p.securityContext.OAuthClientId
	c.Config.ClientSecret = p.securityContext.SharedSecret
	c.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	c.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	return jira.NewClient(c.Client(context.Background()), c.BaseURL)
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
