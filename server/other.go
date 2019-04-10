// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) servePublicKey(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get("Mattermost-User-Id")
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

func (p *Plugin) CreateBotDMPost(userID, message, postType string) *model.AppError {
	channel, aerr := p.API.GetDirectChannel(userID, p.botUserID)
	if aerr != nil {
		p.errorf("Couldn't get bot's DM channel to userId:%v, error:%v", userID, aerr.Error())
		return aerr
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

	_, aerr = p.API.CreatePost(post)
	if aerr != nil {
		p.errorf("Couldn't create post, error:%v", aerr.Error())
		return aerr
	}

	return nil
}

func (p *Plugin) loadJIRAProjectKeys(forceReload bool) ([]string, error) {
	if len(p.projectKeys) > 0 && !forceReload {
		return p.projectKeys, nil
	}

	jiraClient, err := p.getJIRAClientForServer()
	if err != nil {
		return nil, errors.WithMessage(err, "Error connecting to JIRA")
	}

	list, _, err := jiraClient.Project.GetList()
	if err != nil {
		return nil, errors.WithMessage(err, "Error requesting list of JIRA projects")
	}
	keys := []string{}
	for _, proj := range *list {
		keys = append(keys, proj.Key)
	}

	p.projectKeys = keys
	return keys, nil
}
