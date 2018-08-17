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

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	jira "github.com/andygrunwald/go-jira"
	jwt "golang.org/x/oauth2/jira"
)

const (
	JIRA_USERNAME              = "Jira Plugin"
	JIRA_ICON_URL              = "https://s3.amazonaws.com/mattermost-plugin-media/jira.jpg"
	KEY_SECURITY_CONTEXT       = "security_context"
	KEY_USER_INFO              = "user_info_"
	KEY_JIRA_USER_TO_MM_USERID = "jira_user_"
)

type Plugin struct {
	plugin.MattermostPlugin

	Enabled  bool
	Secret   string
	UserName string

	botUserID       string
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

func (p *Plugin) OnActivate() error {
	user, err := p.API.GetUserByUsername(p.UserName)
	if err != nil {
		mlog.Error(err.Error())
		return fmt.Errorf("Unable to find user with configured username: %v", p.UserName)
	}

	p.botUserID = user.Id
	return nil
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
	Name      string
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

	jiraClient, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		http.Error(w, "could not get jira client, err="+err.Error(), 500)
	}

	user, _, err := jiraClient.User.GetSelf()
	if err != nil {
		http.Error(w, "could not get the user, err="+err.Error(), 500)
	}

	info.Name = user.Name

	b, _ := json.Marshal(info)

	p.API.KVSet(KEY_USER_INFO+userID, b)
	p.API.KVSet(KEY_JIRA_USER_TO_MM_USERID+info.Name, []byte(userID))

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

	user, _, err := jiraClient.Issue.GetCreateMeta("")
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

func (p *Plugin) CreateBotDMPost(userID, message, postType string) *model.AppError {
	channel, err := p.API.GetDirectChannel(userID, p.botUserID)
	if err != nil {
		mlog.Error("Couldn't get bot's DM channel", mlog.String("user_id", userID))
		return err
	}

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": JIRA_USERNAME,
			"override_icon_url": JIRA_ICON_URL,
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		mlog.Error(err.Error())
		return err
	}

	return nil
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
		return
	}

	p.handleNotifications(&webhook)

	/*
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
	*/
}

func (p *Plugin) handleNotifications(w *Webhook) {

	switch w.WebhookEvent {
	case "jira:issue_updated":
		p.handleIssueUpdatedNotifications(w)
	case "comment_created":
		p.handleCommentCreatedNotifications(w)
	}
}

// Notify a user when they are assigned to an existing issue
func (p *Plugin) handleIssueUpdatedNotifications(w *Webhook) {
	change := w.ChangeLog.Items[0]
	if change.Field != "assignee" || change.ToString == "" {
		return
	}

	if w.Issue.Fields.Assignee == nil {
		return
	}

	assignee := w.Issue.Fields.Assignee.Name
	if w.User.Name == assignee {
		return
	}

	b, _ := p.API.KVGet(KEY_JIRA_USER_TO_MM_USERID + assignee)
	if b == nil {
		return
	}

	userID := string(b)
	issueURL := getIssueURL(w.Issue)
	userURL := getUserURL(w.Issue, w.User)

	message := "[%s](%s) assigned you to [%s](%s)"

	p.CreateBotDMPost(userID, fmt.Sprintf(message, w.User.DisplayName, userURL, w.Issue.Key, issueURL), "custom_jira_assigned")
}

func (p *Plugin) handleCommentCreatedNotifications(w *Webhook) {
	p.handleCommentMentions(w)

	if w.Issue.Fields.Assignee == nil {
		return
	}

	assignee := w.Issue.Fields.Assignee.Name
	if assignee == w.Comment.Author.Name {
		return
	}

	b, _ := p.API.KVGet(KEY_JIRA_USER_TO_MM_USERID + assignee)
	if b == nil {
		return
	}

	userID := string(b)
	issueURL := getIssueURL(w.Issue)
	userURL := getUserURL(w.Issue, w.Comment.Author)

	message := "[%s](%s) commented on [%s](%s):\n>%s"

	p.CreateBotDMPost(userID, fmt.Sprintf(message, w.Comment.Author.DisplayName, userURL, w.Issue.Key, issueURL, w.Comment.Body), "custom_jira_comment")
}

func (p *Plugin) handleCommentMentions(w *Webhook) {
	mentions := parseJiraUsernamesFromText(w.Comment.Body)

	message := "[%s](%s) mentioned you on [%s](%s):\n>%s"
	issueURL := getIssueURL(w.Issue)

	for _, username := range mentions {
		// Don't notify users of their own comments
		if username == w.Comment.Author.Name {
			continue
		}

		// Notifications for issue assignees are handled separately
		if w.Issue.Fields.Assignee != nil && username == w.Issue.Fields.Assignee.Name {
			continue
		}

		fmt.Println(username)

		b, _ := p.API.KVGet(KEY_JIRA_USER_TO_MM_USERID + username)
		if b == nil {
			continue
		}

		userID := string(b)
		userURL := getUserURL(w.Issue, w.Comment.Author)

		p.CreateBotDMPost(userID, fmt.Sprintf(message, w.Comment.Author.DisplayName, userURL, w.Issue.Key, issueURL, w.Comment.Body), "custom_jira_mention")
	}
}
