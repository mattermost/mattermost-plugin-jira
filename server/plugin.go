// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"text/template"

	jira "github.com/andygrunwald/go-jira"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	JIRA_USERNAME = "Jira Plugin"
	JIRA_ICON_URL = "https://s3.amazonaws.com/mattermost-plugin-media/jira.jpg"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	oauth2Config oauth2.Config

	botUserID   string
	rsaKey      *rsa.PrivateKey
	projectKeys []string

	atlassianConnectTemplate *template.Template
	userConfigTemplate       *template.Template
}

func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()
	user, apperr := p.API.GetUserByUsername(config.UserName)
	if apperr != nil {
		return fmt.Errorf("Unable to find user with configured username: %v, error: %v", config.UserName, apperr)
	}

	bpath, err := p.API.GetBundlePath()
	if err != nil {
		return err
	}

	fpath := filepath.Join(bpath, "server", "dist", "templates", "atlassian-connect.json")
	p.atlassianConnectTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return err
	}

	fpath = filepath.Join(bpath, "server", "dist", "templates", "user-config.html")
	p.userConfigTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return err
	}

	p.botUserID = user.Id
	p.rsaKey = p.getRSAKey()

	// Temporary hack until we can pull the project keys dynamically
	p.projectKeys = []string{"MM"}

	p.oauth2Config = oauth2.Config{
		ClientID:     "LimAAPOhX7ncIN7cPB77tZ1Gwz0r2WmL",
		ClientSecret: "01_Y6g1JRmLnSGcaRU19LzhfnsXHAGwtuQTacQscxR3eCy7tzhLYYbuQHXiVIJq_",
		Scopes:       []string{"read:jira-work", "read:jira-user", "write:jira-work"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.com/authorize",
			TokenURL: "https://auth.atlassian.com/oauth/token",
		},
		RedirectURL: fmt.Sprintf("%v/plugins/%v/oauth/complete", p.externalURL(), manifest.Id),
	}

	p.API.RegisterCommand(getCommand())

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

func (p *Plugin) externalURL() string {
	config := p.getConfiguration()
	if config.ExternalURL != "" {
		return config.ExternalURL
	}
	return *p.API.GetConfig().ServiceSettings.SiteURL
}

func (p *Plugin) serveTest(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	// info, err := p.getJiraUserInfo(userID)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }
	//
	// jiraClient, _, err := p.getJIRAClientForUser(info.AccountId)
	// if err != nil {
	// 	return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	// }
	//
	// user, _, err := jiraClient.Issue.GetCreateMeta("")
	// if err != nil {
	// 	return http.StatusInternalServerError, fmt.Errorf("could not get metadata: %v", err)
	// }
	//
	// userBytes, _ := json.Marshal(user)
	// w.Header().Set("Content-Type", "application/json")
	// w.Write(userBytes)
	return http.StatusOK, nil
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

func (p *Plugin) fetchJIRAProjectKeys(w http.ResponseWriter, r *http.Request) error {
	jiraClient, err := p.getJIRAClientForServer()
	if err != nil {
		return err
	}

	req, _ := jiraClient.NewRawRequest(http.MethodGet, "/rest/api/2/project", nil)
	list1 := jira.ProjectList{}
	_, err = jiraClient.Do(req, &list1)
	if err != nil {
		return err
	}

	list, _, err := jiraClient.Project.GetList()
	if err != nil {
		return err
	}
	keys := []string{}
	for _, proj := range *list {
		keys = append(keys, proj.Key)
	}

	p.projectKeys = keys
	return nil
}
