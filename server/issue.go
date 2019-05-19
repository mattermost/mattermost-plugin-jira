// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func httpAPICreateIssue(a *Action) error {
	api := a.Plugin.API

	create := &struct {
		PostId string           `json:"post_id"`
		Fields jira.IssueFields `json:"fields"`
	}{}
	err := json.NewDecoder(a.HTTPRequest.Body).Decode(&create)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err,
			"failed to decode incoming request")
	}

	// Lets add a permalink to the post in the Jira Description
	post, appErr := api.GetPost(create.PostId)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load post %q", create.PostId)
	}
	if post == nil {
		return a.RespondError(http.StatusInternalServerError, nil,
			"failed to load post %q: not found", create.PostId)
	}
	if channel, _ := api.GetChannel(post.ChannelId); channel != nil {
		if team, _ := api.GetTeam(channel.TeamId); team != nil {
			permalink := fmt.Sprintf("%v/%v/pl/%v",
				a.Plugin.GetSiteURL(),
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

	created, resp, err := a.JiraClient.Issue.Create(&jira.Issue{
		Fields: &create.Fields,
	})
	if err != nil {
		message := "failed to create the issue, postId: " + create.PostId
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return a.RespondError(http.StatusInternalServerError, err, message)
	}

	// Upload file attachments in the background
	if len(post.FileIds) > 0 {
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
				_, _, e := a.JiraClient.Issue.PostAttachment(created.ID, bytes.NewReader(byteData), info.Name)
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
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", a.Instance.GetURL(), created.Key),
		ChannelId: post.ChannelId,
		RootId:    create.PostId,
		UserId:    a.MattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to create notification post: %q", create.PostId)
	}

	return a.RespondJSON(created)
}

func httpAPIGetCreateIssueMetadata(a *Action) error {
	cimd, err := getCreateIssueMetadata(a.JiraClient)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	return a.RespondJSON(cimd)
}

func getCreateIssueMetadata(jiraClient *jira.Client) (*jira.CreateMetaInfo, error) {
	cimd, resp, err := jiraClient.Issue.GetCreateMetaWithOptions(&jira.GetQueryOptions{
		Expand: "projects.issuetypes.fields",
	})
	if err != nil {
		message := "failed to get CreateIssue metadata"
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return nil, errors.WithMessage(err, message)
	}
	return cimd, nil
}

func transitionJiraIssue(a *Action, issueKey, toState string) error {
	transitions, _, err := a.JiraClient.Issue.GetTransitions(issueKey)
	if err != nil {
		return errors.New("We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue.")
	}

	if len(transitions) < 1 {
		return errors.New("You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.")
	}

	var transitionToUse *jira.Transition
	availableStates := []string{}
	for _, transition := range transitions {
		if strings.Contains(strings.ToLower(transition.To.Name), strings.ToLower(toState)) {
			transitionToUse = &transition
		}
		availableStates = append(availableStates, transition.To.Name)
	}

	if transitionToUse == nil {
		return errors.Errorf("%q is not a valid state. Please use one of: %q",
			toState, strings.Join(availableStates, ","))
	}

	if _, err := a.JiraClient.Issue.DoTransition(issueKey, transitionToUse.ID); err != nil {
		return err
	}

	return nil
}
