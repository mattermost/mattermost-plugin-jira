package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

	"bytes"
	jira "github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	jwt "github.com/rbriski/atlassian-jwt"
	oauth2 "golang.org/x/oauth2/jira"
	"github.com/google/go-querystring/query"
)

const (
	JIRA_USERNAME              = "Jira Plugin"
	JIRA_ICON_URL              = "https://s3.amazonaws.com/mattermost-plugin-media/jira.jpg"
	KEY_SECURITY_CONTEXT       = "security_context"
	KEY_USER_INFO              = "user_info_"
	KEY_JIRA_USER_TO_MM_USERID = "jira_user_"
	KEY_RSA                    = "rsa_key"
	KEY_OAUTH1_REQUEST         = "oauth1_request_"
)

type Plugin struct {
	plugin.MattermostPlugin

	Enabled  bool
	Secret   string
	UserName string
	JiraURL  string

	botUserID       string
	securityContext *SecurityContext
	rsaKey          *rsa.PrivateKey
	oauth1Config    *oauth1.Config
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

	p.API.RegisterCommand(getCommand())

	p.botUserID = user.Id
	p.rsaKey = p.getRSAKey()

	p.oauth1Config = &oauth1.Config{
		ConsumerKey:    "OauthKey",
		ConsumerSecret: "dont_care",
		CallbackURL:    *p.API.GetConfig().ServiceSettings.SiteURL + "/plugins/jira/oauth/complete",
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: p.JiraURL + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    p.JiraURL + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  p.JiraURL + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: p.rsaKey},
	}

	return nil
}

func (p *Plugin) getRSAKey() *rsa.PrivateKey {
	b, _ := p.API.KVGet(KEY_RSA)
	if b != nil {
		var key rsa.PrivateKey
		if err := json.Unmarshal(b, &key); err != nil {
			fmt.Println(err.Error())
			return nil
		}
		return &key
	}

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	b, _ = json.Marshal(key)
	p.API.KVSet(KEY_RSA, b)

	return key
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
	case "/public-key":
		p.servePublicKey(w, r)
		return
	case "/oauth/connect":
		p.serveOAuthRequest(w, r)
		return
	case "/oauth/complete":
		p.serveOAuthComplete(w, r)
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
	case "/create-issue":
		p.serveCreateIssue(w, r)
		return
	case "/create-issue-metadata":
		p.serveCreateIssueMetadata(w, r)
		return
	}

	http.NotFound(w, r)
}

func (p *Plugin) serveOAuthRequest(w http.ResponseWriter, r *http.Request) {
	requestToken, requestSecret, err := p.oauth1Config.RequestToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := p.API.KVSet(KEY_OAUTH1_REQUEST+requestToken, []byte(requestSecret)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	authURL, err := p.oauth1Config.AuthorizationURL(requestToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (p *Plugin) serveOAuthComplete(w http.ResponseWriter, r *http.Request) {
	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	requestSecret := ""
	if b, err := p.API.KVGet(KEY_OAUTH1_REQUEST + requestToken); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		requestSecret = string(b)
	}

	p.API.KVDelete(KEY_OAUTH1_REQUEST + requestToken)

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	accessToken, accessSecret, err := p.oauth1Config.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := p.oauth1Config.Client(oauth1.NoContext, token)

	info := &JiraUserInfo{}

	jiraClient, err := jira.NewClient(httpClient, p.JiraURL)
	if err != nil {
		http.Error(w, "could not get jira client, err="+err.Error(), 500)
	}

	req, _ := jiraClient.NewRequest("GET", "rest/api/2/myself", nil)

	data := map[string]interface{}{}
	_, err = jiraClient.Do(req, &data)
	if err != nil {
		http.Error(w, "could not get the user, err="+err.Error(), 500)
	}

	info.AccountId = data["accountId"].(string)
	info.Name = data["name"].(string)

	b, _ := json.Marshal(info)

	p.API.KVSet(KEY_USER_INFO+userID, b)
	p.API.KVSet(KEY_JIRA_USER_TO_MM_USERID+info.Name, []byte(userID))

	html := `
<!DOCTYPE html>
<html>
	<head>
		<script>
			window.close();
		</script>
	</head>
	<body>
		<p>Completed connecting to JIRA.</p>
	</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

type JiraUserInfo struct {
	AccountId string
	Name      string
}

type CreateIssue struct {
	PostId string           `json:"post_id"`
	Fields jira.IssueFields `json:"fields"`
}

func (p *Plugin) servePublicKey(w http.ResponseWriter, r *http.Request) {

	b, err := x509.MarshalPKIXPublicKey(&p.rsaKey.PublicKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pemkey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(pem.EncodeToMemory(pemkey))
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

func (p *Plugin) serveCreateIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cr *CreateIssue
	err := json.NewDecoder(r.Body).Decode(&cr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	info, err := p.getJiraUserInfo(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	jiraClient, _, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		http.Error(w, "could not get jira client, err="+err.Error(), 500)
	}

	// Lets add a permalink to the post in the Jira Description
	description := cr.Fields.Description
	post, _ := p.API.GetPost(cr.PostId)
	if channel, _ := p.API.GetChannel(post.ChannelId); channel != nil {
		if team, _ := p.API.GetTeam(channel.TeamId); team != nil {
			config := p.API.GetConfig()
			baseURL := *config.ServiceSettings.SiteURL
			permalink := fmt.Sprintf("%v/%v/pl/%v",
				baseURL,
				team.Name,
				cr.PostId,
			)

			if len(cr.Fields.Description) > 0 {
				cr.Fields.Description += fmt.Sprintf("\n%v", permalink)
			} else {
				cr.Fields.Description = permalink
			}
		}
	}

	issue := &jira.Issue{
		Fields: &cr.Fields,
	}

	created, _, err := jiraClient.Issue.Create(issue)
	if err != nil {
		http.Error(w, "could not create the issue on Jira, err="+err.Error(), 500)
	}

	// In case the post message is different than the description
	if post != nil &&
		post.UserId == userID &&
		post.Message != description &&
		len(description) > 0 {
		post.Message = description
		p.API.UpdatePost(post)
	}

	if post != nil && len(post.FileIds) > 0 {
		go func() {
			for _, fileId := range post.FileIds {
				info, err := p.API.GetFileInfo(fileId)
				if err == nil {
					byteData, err := p.API.ReadFileAtPath(info.Path)
					if err != nil {
						return
					}
					jiraClient.Issue.PostAttachment(created.ID, bytes.NewReader(byteData), info.Name)
				}
			}
		}()
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message: fmt.Sprintf("Created a Jira issue %v/browse/%v", p.securityContext.BaseURL, created.Key),
		ChannelId: post.ChannelId,
		RootId: cr.PostId,
		UserId: userID,
	}
	p.API.CreatePost(reply)

	userBytes, _ := json.Marshal(created)
	w.Header().Set("Content-Type", "application/json")
	w.Write(userBytes)
}

func (p *Plugin) serveCreateIssueMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	info, err := p.getJiraUserInfo(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	jiraClient, client, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		http.Error(w, "could not get jira client, err="+err.Error(), 500)
	}

	var metadata []byte
	options := &jira.GetQueryOptions{ProjectKeys: "", Expand: "projects.issuetypes.fields"}
	req, _ := jiraClient.NewRawRequest("GET", "rest/api/2/issue/createmeta", nil)

	if options != nil {
		q, err := query.Values(options)
		if err != nil {
			http.Error(w, "could not get the create issue metadata from Jira, err="+err.Error(), 500)
			return
		}
		req.URL.RawQuery = q.Encode()
	}

	httpResp, err := client.Do(req)
	if err != nil {
		http.Error(w, "could not get the create issue metadata from Jira in request, err="+err.Error(), 500)
		return
	} else {
		defer httpResp.Body.Close()
		metadata, _ = ioutil.ReadAll(httpResp.Body)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metadata)
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

	jiraClient, _, err := p.getJIRAClientForUser(info.AccountId)
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

func (p *Plugin) getJIRAClientForUser(jiraUser string) (*jira.Client, *http.Client, error) {
	if p.securityContext == nil {
		p.loadSecurityContext()
	}

	c := oauth2.Config{
		BaseURL: p.securityContext.BaseURL,
		Subject: jiraUser,
	}

	c.Config.ClientID = p.securityContext.OAuthClientId
	c.Config.ClientSecret = p.securityContext.SharedSecret
	c.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	c.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := c.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, c.BaseURL)
	return jiraClient, httpClient, err
}

func (p *Plugin) getJIRAClientForServer() (*jira.Client, error) {
	if p.securityContext == nil {
		p.loadSecurityContext()
	}

	c := &jwt.Config{
		Key:          p.securityContext.Key,
		ClientKey:    p.securityContext.ClientKey,
		SharedSecret: p.securityContext.SharedSecret,
		BaseURL:      p.securityContext.BaseURL,
	}

	return jira.NewClient(c.Client(), c.BaseURL)
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
	if w.Comment.Author.Name == "addon_mattermost-jira-plugin" {
		return
	}

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

		b, _ := p.API.KVGet(KEY_JIRA_USER_TO_MM_USERID + username)
		if b == nil {
			continue
		}

		userID := string(b)
		userURL := getUserURL(w.Issue, w.Comment.Author)

		p.CreateBotDMPost(userID, fmt.Sprintf(message, w.Comment.Author.DisplayName, userURL, w.Issue.Key, issueURL, w.Comment.Body), "custom_jira_mention")
	}
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	issues := parseJiraIssueFromText(post.Message)
	if len(issues) == 0 {
		return
	}

	jiraClient, _ := p.getJIRAClientForServer()

	config := p.API.GetConfig()

	channel, _ := p.API.GetChannel(post.ChannelId)
	if channel == nil {
		return
	}

	if channel.Type != model.CHANNEL_OPEN {
		return
	}

	team, _ := p.API.GetTeam(channel.TeamId)
	if team == nil {
		return
	}

	user, _ := p.API.GetUser(post.UserId)
	if user == nil {
		return
	}

	for _, issue := range issues {
		permalink := *config.ServiceSettings.SiteURL + "/" + team.Name + "/pl/" + post.Id

		comment := &jira.Comment{
			Body: fmt.Sprintf("%s mentioned this ticket in Mattermost:\n{quote}\n%s\n{quote}\n\n[View message in Mattermost|%s]", user.Username, post.Message, permalink),
		}

		_, resp, err := jiraClient.Issue.AddComment(issue, comment)
		if err != nil {
			fmt.Println(resp.StatusCode)
			fmt.Println(err.Error())
		}
	}
}
