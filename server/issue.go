// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func httpAPICreateIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be POST")
	}

	api := ji.GetPlugin().API

	create := &struct {
		PostId         string           `json:"post_id"`
		AttachedToPost bool             `json:"attached_to_post"`
		ChannelId      string           `json:"channel_id"`
		Fields         jira.IssueFields `json:"fields"`
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

	jiraUser, err := ji.GetPlugin().LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var post *model.Post
	var appErr *model.AppError

	// If this issue is attached to a post, lets add a permalink to the post in the Jira Description
	if create.AttachedToPost {
		post, appErr = api.GetPost(create.PostId)
		if appErr != nil {
			return http.StatusInternalServerError,
				errors.WithMessage(appErr, "failed to load post "+create.PostId)
		}
		if post == nil {
			return http.StatusInternalServerError,
				errors.New("failed to load post " + create.PostId + ": not found")
		}
		if channel, _ := api.GetChannel(post.ChannelId); channel != nil {
			if team, _ := api.GetTeam(channel.TeamId); team != nil {
				permalink := fmt.Sprintf("%v/%v/pl/%v",
					ji.GetPlugin().GetSiteURL(),
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
	}

	created, _, err := jiraClient.Issue.Create(&jira.Issue{
		Fields: &create.Fields,
	})

	// For now, if we are not attaching to a post, just postId blank (this will only affect the error message)
	var channelId, postId string
	if create.AttachedToPost {
		channelId = post.ChannelId
		postId = create.PostId
	} else {
		channelId = create.ChannelId
		postId = ""
	}

	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to create the issue, postId (if attached to a post): "+postId)
	}

	// Upload file attachments in the background
	if create.AttachedToPost && len(post.FileIds) > 0 {
		go func() {
			for _, fileId := range post.FileIds {
				info, ae := api.GetFileInfo(fileId)
				if ae != nil {
					continue
				}
				// TODO: large file support? Ignoring errors for now is good enough...
				byteData, ae := api.ReadFile(info.Path)
				if ae != nil {
					// TODO report errors, as DMs from JIRA bot?
					api.LogError("failed to attach file to issue: "+ae.Error(), "file", info.Path, "issue", created.Key)
					return
				}
				_, _, e := jiraClient.Issue.PostAttachment(created.ID, bytes.NewReader(byteData), info.Name)
				if e != nil {
					// TODO report errors, as DMs from JIRA bot?
					api.LogError("failed to attach file to issue: "+e.Error(), "file", info.Path, "issue", created.Key)
					return
				}

			}
		}()
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		// TODO: Why is this not created.Self?
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", ji.GetURL(), created.Key),
		ChannelId: channelId,
		RootId:    postId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post, postId (if attached to a post): "+postId)
	}

	userBytes, err := json.Marshal(created)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to marshal response, postId (if attached to a post) "+postId)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(userBytes)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response, postId (if attached to a post) "+postId)
	}
	return http.StatusOK, nil
}

func httpAPIGetCreateIssueMetadata(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("Request: " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	jiraUser, err := ji.GetPlugin().LoadJIRAUser(ji, mattermostUserId)
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

func (p *Plugin) transitionJiraIssue(mmUserId, issueKey, toState string) error {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return err
	}

	jiraUser, err := ji.GetPlugin().LoadJIRAUser(ji, mmUserId)
	if err != nil {
		return err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return err
	}

	transitions, _, err := jiraClient.Issue.GetTransitions(issueKey)
	if err != nil {
		return fmt.Errorf("We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue.")
	}

	if len(transitions) < 1 {
		return fmt.Errorf("You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.")
	}

	var transitionToUse *jira.Transition
	for _, transition := range transitions {
		if strings.Contains(strings.ToLower(transition.To.Name), strings.ToLower(toState)) {
			transitionToUse = &transition
			break
		}
	}

	if transitionToUse == nil {
		return fmt.Errorf("We couldn't find the state. Please use a Jira state such as 'done' and try again.")
	}

	if _, err := jiraClient.Issue.DoTransition(issueKey, transitionToUse.ID); err != nil {
		return err
	}

	return nil
}
