// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	jira "github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/model"
)

type MattermostCreateIssueRequest struct {
	PostId string           `json:"post_id"`
	Fields jira.IssueFields `json:"fields"`
}

func httpAPICreateIssue(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}

	create := &MattermostCreateIssueRequest{}
	err := json.NewDecoder(r.Body).Decode(&create)
	if err != nil {
		return http.StatusBadRequest, err
	}

	mmUserID := r.Header.Get("Mattermost-User-Id")
	if mmUserID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser, err := p.LoadJIRAUser(ji, mmUserID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	}

	// Lets add a permalink to the post in the Jira Description
	description := create.Fields.Description
	post, _ := p.API.GetPost(create.PostId)
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

	issue := &jira.Issue{
		Fields: &create.Fields,
	}

	created, _, err := jiraClient.Issue.Create(issue)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not create issue in jira: %v", err)
	}

	// In case the post message is different than the description
	if post != nil && post.UserId == mmUserID && post.Message != description && len(description) > 0 {
		post.Message = description
		p.API.UpdatePost(post)
	}

	if post != nil && len(post.FileIds) > 0 {
		go func() {
			for _, fileId := range post.FileIds {
				info, aerr := p.API.GetFileInfo(fileId)
				if aerr == nil {
					byteData, aerr := p.API.ReadFile(info.Path)
					if aerr != nil {
						return
					}
					jiraClient.Issue.PostAttachment(created.ID, bytes.NewReader(byteData), info.Name)
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
		UserId:    mmUserID,
	}
	p.API.CreatePost(reply)

	userBytes, _ := json.Marshal(created)
	w.Header().Set("Content-Type", "application/json")
	w.Write(userBytes)
	return http.StatusOK, nil
}

func httpAPIGetCreateIssueMetadata(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}

	mmUserID := r.Header.Get("Mattermost-User-Id")
	if mmUserID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraUser, err := p.LoadJIRAUser(ji, mmUserID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	}

	cimd, _, err := jiraClient.Issue.GetCreateMetaWithOptions(&jira.GetQueryOptions{
		Expand: "projects.issuetypes.fields",
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not get issue metadata: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(cimd)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not marshal CreateIssueMetadata: %v", err)
	}
	_, err = w.Write(b)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not write output: %v", err)
	}

	return http.StatusOK, nil
}
