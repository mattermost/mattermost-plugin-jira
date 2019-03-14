// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	// "bytes"
	// "crypto/rand"
	// "crypto/rsa"
	// "crypto/x509"
	"encoding/json"
	// "encoding/pem"
	"fmt"
	// "io/ioutil"
	"net/http"
	// "sync"

	jira "github.com/andygrunwald/go-jira"
	// "github.com/google/go-querystring/query"
	// "golang.org/x/oauth2"
	// "github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	// "github.com/mattermost/mattermost-server/plugin"
)

type CreateIssue struct {
	PostId string           `json:"post_id"`
	Fields jira.IssueFields `json:"fields"`
}

func (p *Plugin) handleHTTPCreateIssue(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}

	cr := &CreateIssue{}
	err := json.NewDecoder(r.Body).Decode(&cr)
	if err != nil {
		return http.StatusBadRequest, err
	}

	mmUserID := r.Header.Get("Mattermost-User-ID")
	if mmUserID == "" {
		return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	}

	// info, err := p.getJiraUserInfo(userID)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }

	jiraUserID := ""

	jiraClient, _, err := p.getJIRAClientForUser(jiraUserID)
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
	if post != nil && post.UserId == mmUserID && post.Message != description && len(description) > 0 {
		post.Message = description
		p.API.UpdatePost(post)
	}

	// if post != nil && len(post.FileIds) > 0 {
	// 	go func() {
	// 		for _, fileId := range post.FileIds {
	// 			info, err := p.API.GetFileInfo(fileId)
	// 			if err == nil {
	// 				byteData, err := p.API.ReadFile(info.Path)
	// 				if err != nil {
	// 					return
	// 				}
	// 				jiraClient.Issue.PostAttachment(created.ID, bytes.NewReader(byteData), info.Name)
	// 			}
	// 		}
	// 	}()
	// }
	//
	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", p.sc.BaseURL, created.Key),
		ChannelId: post.ChannelId,
		RootId:    cr.PostId,
		UserId:    mmUserID,
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

	// userID := r.Header.Get("Mattermost-User-ID")
	// if userID == "" {
	// 	return http.StatusUnauthorized, fmt.Errorf("Not authorized")
	// }
	//
	// info, err := p.getJiraUserInfo(userID)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }
	//
	// jiraClient, client, err := p.getJIRAClientForUser(info.AccountId)
	// if err != nil {
	// 	return http.StatusInternalServerError, fmt.Errorf("could not get jira client: %v", err)
	// }
	//
	// var metadata []byte
	// options := &jira.GetQueryOptions{ProjectKeys: "", Expand: "projects.issuetypes.fields"}
	// req, _ := jiraClient.NewRawRequest("GET", "rest/api/2/issue/createmeta", nil)
	//
	// if options != nil {
	// 	q, err := query.Values(options)
	// 	if err != nil {
	// 		return http.StatusInternalServerError, fmt.Errorf("could not get the create issue metadata from Jira: %v", err)
	// 	}
	// 	req.URL.RawQuery = q.Encode()
	// }
	// httpResp, err := client.Do(req)
	// if err != nil {
	// 	return http.StatusInternalServerError, fmt.Errorf("could not get the create issue metadata from Jira in request: %v", err)
	// } else {
	// 	defer httpResp.Body.Close()
	// 	metadata, _ = ioutil.ReadAll(httpResp.Body)
	// }
	//
	// w.Header().Set("Content-Type", "application/json")
	// w.Write(metadata)
	return http.StatusOK, nil
}
