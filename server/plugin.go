// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	// "crypto/subtle"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	// "net/url"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	jira "github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/google/go-querystring/query"
	jwt "github.com/rbriski/atlassian-jwt"
	oauth2 "golang.org/x/oauth2/jira"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	JIRA_USERNAME              = "Jira Plugin"
	JIRA_ICON_URL              = "https://s3.amazonaws.com/mattermost-plugin-media/jira.jpg"
	KEY_SECURITY_CONTEXT       = "security_context"
	KEY_USER_INFO              = "user_info_"
	KEY_JIRA_USER_TO_MM_USERID = "jira_user_"
	KEY_RSA                    = "rsa_key"
	KEY_OAUTH1_REQUEST         = "oauth1_request_"
	WS_EVENT_CONNECT           = "connect"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// SecurityContext is provided by JIRA upon the installation of this integration
	// on the JIRA side. We store it in the DB and refresh as needed
	sc           SecurityContext
	oauth1Config *oauth1.Config

	botUserID   string
	rsaKey      *rsa.PrivateKey
	projectKeys []string
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
	var err error
	err = p.loadSecurityContext()
	if err != nil {
		p.API.LogInfo("Failed to load the security context to connect to JIRA. Make sure you install on the JIRA side\n")
	}
	fmt.Printf("<><> 2 OnActivate: err: %v, sc:%#v\n", err, p.sc)

	config := p.getConfiguration()
	user, apperr := p.API.GetUserByUsername(config.UserName)
	if apperr != nil {
		return fmt.Errorf("Unable to find user with configured username: %v, error: %v", config.UserName, apperr)
	}

	p.botUserID = user.Id
	p.rsaKey = p.getRSAKey()

	// Temporary hack until we can pull the project keys dynamically
	p.projectKeys = []string{"MM"}

	p.oauth1Config = &oauth1.Config{
		ConsumerKey:    "OauthKey",
		ConsumerSecret: "dont_care",
		CallbackURL:    p.externalURL() + "/plugins/" + manifest.Id + "/oauth/complete",
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: p.sc.BaseURL + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    p.sc.BaseURL + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  p.sc.BaseURL + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: p.rsaKey},
	}

	p.API.RegisterCommand(getCommand())

	return nil
}

func (p *Plugin) loadSecurityContext() error {
	// Since .sc is not a pointer, use .Key to check if it's already loaded
	if p.sc.Key != "" {
		return nil
	}

	b, apperr := p.API.KVGet(KEY_SECURITY_CONTEXT)
	if apperr != nil {
		return apperr
	}
	var sc SecurityContext
	err := json.Unmarshal(b, &sc)
	if err != nil {
		return err
	}
	p.sc = sc
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
	config := p.getConfiguration()
	if config.UserName == "" {
		http.Error(w, "JIRA plugin not configured correctly; must provide UserName", http.StatusForbidden)
		return
	}

	status, err := p.handleHTTPRequest(w, r)
	if err != nil {
		p.API.LogError("ERROR: ", "Status", strconv.Itoa(status), "Error", err.Error(), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
		http.Error(w, err.Error(), status)
	}
	p.API.LogDebug("OK: ", "Status", strconv.Itoa(status), "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method, "query", r.URL.Query().Encode())
}

func (p *Plugin) handleHTTPRequest(w http.ResponseWriter, r *http.Request) (int, error) {
	switch r.URL.Path {
	case "/test":
		return p.serveTest(w, r)
	case "/public-key":
		return p.servePublicKey(w, r)
	case "/oauth/connect":
		return p.serveOAuthRequest(w, r)
	case "/oauth/complete":
		return p.serveOAuthComplete(w, r)
	case "/webhook",
		"/issue_event":
		return p.serveWebhook(w, r)
	case "/atlassian-connect-jwt.json":
		return p.serveAtlassianConnectJWT(w, r)
	case "/atlassian-connect-oauth.json":
		return p.serveAtlassianConnectOauth(w, r)
	case "/installed":
		return p.serveInstalled(w, r)
	case "/uninstalled":
		return p.serveUninstalled(w, r)
	case "/create-issue":
		return p.serveCreateIssue(w, r)
	case "/create-issue-metadata":
		return p.serveCreateIssueMetadata(w, r)
	case "/api/v1/connected":
		return p.serveGetConnected(w, r)
	}

	return http.StatusNotFound, fmt.Errorf("Not found")
}

type ConnectedResponse struct {
	Connected     bool   `json:"connected"`
	JiraUsername  string `json:"jira_username"`
	JiraAccountId string `json:"jira_account_id"`
	JiraURL       string `json:"jira_url"`
}

func (p *Plugin) serveGetConnected(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	resp := &ConnectedResponse{Connected: false, JiraURL: p.sc.BaseURL}

	info, _ := p.getJiraUserInfo(userID)
	if info != nil {
		resp.Connected = true
		resp.JiraUsername = info.Name
		resp.JiraAccountId = info.AccountId
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
	return http.StatusOK, nil
}

func (p *Plugin) serveOAuthRequest(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	requestToken, requestSecret, err := p.oauth1Config.RequestToken()
	fmt.Printf("<><> serveOAuthRequest %v '%v' '%v'", err, requestToken, requestSecret)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// values := url.Values{}
	// values.Add("scope", "read:jira-work")
	// values.Add("client_id", p.sc.OAuthClientId)
	// values.Add("redirect_uri", fmt.Sprintf("%v/plugins/%v/oath/complete", p.externalURL(), manifest.Id))
	// values.Add("state", userID)
	// values.Add("audience", "api.atlassian.com")
	// values.Add("response_type", "code")
	// values.Add("prompt", "consent")
	// linkURL := "https://auth.atlassian.com/authorize?" + values.Encode()
	// fmt.Printf("<><> %q\n", linkURL)

	//http.Redirect(w, r, linkURL, http.StatusFound)
	// return http.StatusFound, nil
	return http.StatusOK, nil
}

func (p *Plugin) serveOAuthComplete(w http.ResponseWriter, r *http.Request) (int, error) {
	bb, err := httputil.DumpRequest(r, true)
	fmt.Printf("<><> serveOAuthComplete %v %v", err, string(bb))
	// requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }
	//
	// requestSecret := ""
	// if b, err := p.API.KVGet(KEY_OAUTH1_REQUEST + requestToken); err != nil {
	// 	return http.StatusInternalServerError, err
	// } else {
	// 	requestSecret = string(b)
	// }
	// p.API.KVDelete(KEY_OAUTH1_REQUEST + requestToken)
	//
	// userID := r.Header.Get("Mattermost-User-ID")
	// if userID == "" {
	// 	return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	// }
	//
	// accessToken, accessSecret, err := p.oauth1Config.AccessToken(requestToken, requestSecret, verifier)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }
	//
	// token := oauth1.NewToken(accessToken, accessSecret)
	// httpClient := p.oauth1Config.Client(oauth1.NoContext, token)
	//
	// info := &JiraUserInfo{}
	//
	// jiraClient, err := jira.NewClient(httpClient, p.sc.BaseURL)
	// if err != nil {
	// 	return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	// }
	//
	// req, _ := jiraClient.NewRequest("GET", "rest/api/2/myself", nil)
	//
	// data := map[string]interface{}{}
	// _, err = jiraClient.Do(req, &data)
	// if err != nil {
	// 	return http.StatusInternalServerError, fmt.Errorf("could not get user: %v", err)
	// }
	//
	// info.AccountId = data["accountId"].(string)
	// info.Name = data["name"].(string)
	//
	// b, _ := json.Marshal(info)
	//
	// p.API.KVSet(KEY_USER_INFO+userID, b)
	// p.API.KVSet(KEY_JIRA_USER_TO_MM_USERID+info.Name, []byte(userID))
	//
	// p.API.PublishWebSocketEvent(
	// 	WS_EVENT_CONNECT,
	// 	map[string]interface{}{
	// 		"connected":       true,
	// 		"jira_username":   info.Name,
	// 		"jira_account_id": info.AccountId,
	// 		"jira_url":        p.sc.BaseURL,
	// 	},
	// 	&model.WebsocketBroadcast{UserId: userID},
	// )
	//
	html := `
<!DOCTYPE html>
<html>
       <head>
               <script>
                       window.close();
               </script>
       </head>
       <body>
               <p>Completed connecting to JIRA. Please close this page.</p>
       </body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
	return http.StatusOK, nil
}

type JiraUserInfo struct {
	AccountId string
	Name      string
}

type CreateIssue struct {
	PostId string           `json:"post_id"`
	Fields jira.IssueFields `json:"fields"`
}

func (p *Plugin) servePublicKey(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	if !p.API.HasPermissionTo(userID, model.PERMISSION_MANAGE_SYSTEM) {
		return http.StatusForbidden, fmt.Errorf("Forbidden")
	}

	b, err := x509.MarshalPKIXPublicKey(&p.rsaKey.PublicKey)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	pemkey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(pem.EncodeToMemory(pemkey))
	return http.StatusOK, nil
}

func (p Plugin) externalURL() string {
	config := p.getConfiguration()
	if config.ExternalURL != "" {
		return config.ExternalURL
	}
	return *p.API.GetConfig().ServiceSettings.SiteURL
}

func (p *Plugin) serveAtlassianConnectJWT(w http.ResponseWriter, r *http.Request) (int, error) {
	return p.serveAtlassianConnectJSON(w, r, "jwt")
}

func (p *Plugin) serveAtlassianConnectOauth(w http.ResponseWriter, r *http.Request) (int, error) {
	return p.serveAtlassianConnectJSON(w, r, "oauth")
}

func (p *Plugin) serveAtlassianConnectJSON(w http.ResponseWriter, r *http.Request, authType string) (int, error) {
	baseURL := p.externalURL() + "/" + path.Join("plugins", manifest.Id)

	fmt.Println("<><><><><>", *p.API.GetConfig().PluginSettings.Directory, manifest.Id, "server", "dist", "templates", "atlassian-connect"+authType+".json")
	lp := filepath.Join(*p.API.GetConfig().PluginSettings.Directory, manifest.Id, "server", "dist", "templates", "atlassian-connect-"+authType+".json")
	fmt.Printf("<><> serveAtlassianConnect: %v\n", lp)
	vals := map[string]string{
		"BaseURL": baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	bb := &bytes.Buffer{}
	tmpl.ExecuteTemplate(bb, "config", vals)
	fmt.Printf("<><> serveAtlassianConnect: %v\n", bb.String())
	io.Copy(w, bb)
	return http.StatusOK, nil
}

func (p *Plugin) serveInstalled(w http.ResponseWriter, r *http.Request) (int, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var sc SecurityContext
	err = json.Unmarshal(body, &sc)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	p.sc = sc

	// TODO in a cluster situation, other instances should be notified and re-configure
	// themselves
	appErr := p.API.KVSet(KEY_SECURITY_CONTEXT, body)
	fmt.Printf("<><> SecurityContext payload (%v): %v\n", appErr, string(body))

	// Attempted to auto load the project keys but the jira client was failing for some reason
	// Need to look into it some more later

	/*if jiraClient, _ := p.getJIRAClientForServer(); jiraClient != nil {
	        fmt.Println("HIT0")
	        req, _ := jiraClient.NewRawRequest(http.MethodGet, "/rest/api/2/project", nil)
	        list1 := jira.ProjectList{}
	        _, err1 := jiraClient.Do(req, &list1)
	        if err1 != nil {
	                fmt.Println(err1.Error())
	        }

	        fmt.Println(list1)

	        if list, resp, err := jiraClient.Project.GetList(); err == nil {
	                fmt.Println("HIT1")
	                keys := []string{}
	                for _, proj := range *list {
	                        keys = append(keys, proj.Key)
	                }
	                p.projectKeys = keys
	                fmt.Println(p.projectKeys)
	        } else {
	                body, _ := ioutil.ReadAll(resp.Body)
	                fmt.Println(string(body))
	                fmt.Println(err.Error())
	        }
	}*/

	json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}

func (p *Plugin) serveUninstalled(w http.ResponseWriter, r *http.Request) (int, error) {
	// body, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	log.Fatalf("Can't read request:%v\n", err)
	// 	return http.StatusInternalServerError, err
	// }
	//
	// fmt.Printf("<><> SecurityContext payload: %v\n", string(body))
	//
	json.NewEncoder(w).Encode([]string{"OK"})
	return http.StatusOK, nil
}

func (p *Plugin) serveCreateIssue(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}

	var cr *CreateIssue
	err := json.NewDecoder(r.Body).Decode(&cr)
	if err != nil {
		return http.StatusBadRequest, err
	}

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	info, err := p.getJiraUserInfo(userID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, _, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	}

	// Lets add a permalink to the post in the Jira Description
	description := cr.Fields.Description
	post, _ := p.API.GetPost(cr.PostId)
	if channel, _ := p.API.GetChannel(post.ChannelId); channel != nil {
		if team, _ := p.API.GetTeam(channel.TeamId); team != nil {
			permalink := fmt.Sprintf("%v/%v/pl/%v",
				p.externalURL(),
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
		return http.StatusInternalServerError, fmt.Errorf("could not create issue in jira: %v", err)
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
					byteData, err := p.API.ReadFile(info.Path)
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
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", p.sc.BaseURL, created.Key),
		ChannelId: post.ChannelId,
		RootId:    cr.PostId,
		UserId:    userID,
	}
	p.API.CreatePost(reply)

	userBytes, _ := json.Marshal(created)
	w.Header().Set("Content-Type", "application/json")
	w.Write(userBytes)
	return http.StatusOK, nil
}

func (p *Plugin) serveCreateIssueMetadata(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	info, err := p.getJiraUserInfo(userID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, client, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	}

	var metadata []byte
	options := &jira.GetQueryOptions{ProjectKeys: "", Expand: "projects.issuetypes.fields"}
	req, _ := jiraClient.NewRawRequest("GET", "rest/api/2/issue/createmeta", nil)

	if options != nil {
		q, err := query.Values(options)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("could not get the create issue metadata from Jira: %v", err)
		}
		req.URL.RawQuery = q.Encode()
	}
	httpResp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get the create issue metadata from Jira in request: %v", err)
	} else {
		defer httpResp.Body.Close()
		metadata, _ = ioutil.ReadAll(httpResp.Body)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metadata)
	return http.StatusOK, nil
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

func (p *Plugin) serveTest(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	info, err := p.getJiraUserInfo(userID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, _, err := p.getJIRAClientForUser(info.AccountId)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	}

	user, _, err := jiraClient.Issue.GetCreateMeta("")
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get metadata: %v", err)
	}

	userBytes, _ := json.Marshal(user)
	w.Header().Set("Content-Type", "application/json")
	w.Write(userBytes)
	return http.StatusOK, nil
}

func (p *Plugin) getJIRAClientForUser(jiraUser string) (*jira.Client, *http.Client, error) {
	// TODO Is this redundant?
	err := p.loadSecurityContext()
	if err != nil {
		return nil, nil, err
	}

	c := oauth2.Config{
		BaseURL: p.sc.BaseURL,
		Subject: jiraUser,
	}

	c.Config.ClientID = p.sc.OAuthClientId
	c.Config.ClientSecret = p.sc.SharedSecret
	c.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	c.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := c.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, c.BaseURL)
	return jiraClient, httpClient, err
}

func (p *Plugin) getJIRAClientForServer() (*jira.Client, error) {
	// TODO Is this redundant?
	err := p.loadSecurityContext()
	if err != nil {
		return nil, err
	}

	c := &jwt.Config{
		Key:          p.sc.Key,
		ClientKey:    p.sc.ClientKey,
		SharedSecret: p.sc.SharedSecret,
		BaseURL:      p.sc.BaseURL,
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

func (p *Plugin) serveWebhook(w http.ResponseWriter, r *http.Request) (int, error) {
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
