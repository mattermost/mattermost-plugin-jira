// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func httpAPICreateIssue(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be POST")
	}

	create := &struct {
		PostId string           `json:"post_id"`
		Fields jira.IssueFields `json:"fields"`
	}{}
	err := json.NewDecoder(r.Body).Decode(&create)
	if err != nil {
		return http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser, err := p.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Lets add a permalink to the post in the Jira Description
	post, appErr := p.API.GetPost(create.PostId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load post "+create.PostId)
	}
	if post == nil {
		return http.StatusInternalServerError,
			errors.New("failed to load post " + create.PostId + ": not found")
	}
	if channel, _ := p.API.GetChannel(post.ChannelId); channel != nil {
		if team, _ := p.API.GetTeam(channel.TeamId); team != nil {
			permalink := fmt.Sprintf("%v/%v/pl/%v",
				p.GetSiteURL(),
				team.Name,
				create.PostId,
			)

			if len(create.Fields.Description) > 0 {
				create.Fields.Description += fmt.Sprintf("\n%v", permalink)
			} else {
				create.Fields.Description = permalink
			}
		}
	}

	created, _, err := jiraClient.Issue.Create(&jira.Issue{
		Fields: &create.Fields,
	})
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to create the issue, postId: "+create.PostId)
	}

	// Upload file attachments in the background
	if len(post.FileIds) > 0 {
		go func() {
			for _, fileId := range post.FileIds {
				info, appErr := p.API.GetFileInfo(fileId)
				if appErr == nil {
					// TODO: large file support? Ignoring errors for now is good enough...
					byteData, appErr := p.API.ReadFile(info.Path)
					if appErr != nil {
						// TODO report errors, as DMs from JIRA bot?
						p.errorf("failed to attach file %s to issue %s: %s", info.Path, created.Key, appErr.Error())
						return
					}
					_, _, err := jiraClient.Issue.PostAttachment(created.ID, bytes.NewReader(byteData), info.Name)
					if err != nil {
						// TODO report errors, as DMs from JIRA bot?
						p.errorf("failed to attach file %s to issue %s: %v", info.Path, created.Key, err)
						return
					}
				}
			}
		}()
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		// TODO: Why is this not created.Self?
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", ji.GetURL(), created.Key),
		ChannelId: post.ChannelId,
		RootId:    create.PostId,
		UserId:    mattermostUserId,
	}
	_, appErr = p.API.CreatePost(reply)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post "+create.PostId)
	}

	userBytes, err := json.Marshal(created)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to marshal response "+create.PostId)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(userBytes)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response "+create.PostId)
	}
	return http.StatusOK, nil
}

func httpAPIGetCreateIssueMetadata(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("Request: " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser, err := p.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	cimd, _, err := jiraClient.Issue.GetCreateMetaWithOptions(&jira.GetQueryOptions{
		Expand: "projects.issuetypes.fields",
	})
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to get CreateIssue mettadata")
	}

	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(cimd)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to marshal response")
	}
	_, err = w.Write(b)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}

	return http.StatusOK, nil
}
