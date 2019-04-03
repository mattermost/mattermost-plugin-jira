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

	jira "github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

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
