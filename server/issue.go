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

var httpAPICreateIssue = []ActionFunc{
	RequireHTTPPost,
	RequireMattermostUserId,
	RequireInstance,
	RequireJiraUser,
	RequireJiraClient,
	handleAPICreateIssue,
}

func handleAPICreateIssue(a *Action) error {
	api := a.API

	createRequest := &struct {
		PostId    string           `json:"post_id"`
		ChannelId string           `json:"channel_id"`
		Fields    jira.IssueFields `json:"fields"`
	}{}
	err := json.NewDecoder(a.HTTPRequest.Body).Decode(&createRequest)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err,
			"failed to decode incoming request")
	}

	var post *model.Post
	var appErr *model.AppError

	// If this issue is attached to a post, lets add a permalink to the post in the Jira Description
	if createRequest.PostId != "" {
		post, appErr = api.GetPost(createRequest.PostId)
		if appErr != nil {
			return a.RespondError(http.StatusInternalServerError, appErr,
				"failed to load post %q", createRequest.PostId)
		}
		if post == nil {
			return a.RespondError(http.StatusInternalServerError, nil,
				"failed to load post %q: not found", createRequest.PostId)
		}
		permalink := ""
		permalink, err = getPermaLink(a, createRequest.PostId, post)
		if err != nil {
			return a.RespondError(http.StatusInternalServerError, nil,
				"failed to get permalink for: %q", createRequest.PostId)
		}

		if len(createRequest.Fields.Description) > 0 {
			createRequest.Fields.Description += "\n" + permalink
		} else {
			createRequest.Fields.Description = permalink
		}
	}

	channelId := createRequest.ChannelId
	if post != nil {
		channelId = post.ChannelId
	}

	createdIssue, resp, err := a.JiraClient.Issue.Create(&jira.Issue{
		Fields: &createRequest.Fields,
	})
	if err != nil {
		message := "failed to create the issue, postId: " + createRequest.PostId + ", channelId: " + channelId
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return a.RespondError(http.StatusInternalServerError, err, message)
	}

	// Upload file attachments in the background
	if post != nil && len(post.FileIds) > 0 {
		go func() {
			for _, fileId := range post.FileIds {
				info, ae := api.GetFileInfo(fileId)
				if ae != nil {
					continue
				}
				// TODO: large file support? Ignoring errors for now is good enough...
				byteData, ae := api.ReadFile(info.Path)
				if ae != nil {
					// TODO report errors, as DMs from Jira bot?
					a.Infof("failed to attach file %q to issue %q: %s",
						info.Path, createdIssue.Key, appErr.Error())
					return
				}
				_, _, e := a.JiraClient.Issue.PostAttachment(createdIssue.ID, bytes.NewReader(byteData), info.Name)
				if e != nil {
					// TODO report errors, as DMs from Jira bot?
					a.Infof("failed to attach file %q to issue %q: %s",
						info.Path, createdIssue.Key, appErr.Error())
					return
				}
			}
		}()
	}

	rootId := createRequest.PostId
	parentId := ""
	if post.ParentId != "" {
		// the original post was a reply
		rootId = post.RootId
		parentId = createRequest.PostId
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", a.Instance.GetURL(), createdIssue.Key),
		ChannelId: channelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    a.MattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to create notification post: %q", createRequest.PostId)
	}

	return a.RespondJSON(createdIssue)
}

var httpAPIGetCreateIssueMetadata = []ActionFunc{
	RequireHTTPGet,
	RequireMattermostUserId,
	RequireInstance,
	RequireJiraUser,
	RequireJiraClient,
	handleAPIGetCreateIssueMetadata,
}

func handleAPIGetCreateIssueMetadata(a *Action) error {
	cimd, err := getCreateIssueMetadata(a.JiraClient)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	return a.RespondJSON(cimd)
}

var httpAPIGetSearchIssues = []ActionFunc{
	RequireHTTPGet,
	RequireMattermostUserId,
	RequireInstance,
	RequireJiraUser,
	RequireJiraClient,
	handleAPIGetSearchIssues,
}

func handleAPIGetSearchIssues(a *Action) error {
	jqlString := a.HTTPRequest.FormValue("jql")

	searchRes, resp, err := a.JiraClient.Issue.Search(jqlString, &jira.SearchOptions{
		MaxResults: 50,
		Fields:     []string{"key", "summary"},
	})

	if err != nil {
		message := "failed to get search results"
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details: " + string(bb)
		}
		return a.RespondError(http.StatusInternalServerError, err, message)
	}

	// We only need to send down a summary of the data
	type issueSummary struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	resSummary := make([]issueSummary, 0, len(searchRes))
	for _, res := range searchRes {
		resSummary = append(resSummary, issueSummary{
			Value: res.Key,
			Label: res.Key + ": " + res.Fields.Summary,
		})
	}

	return a.RespondJSON(resSummary)
}

var httpAPIAttachCommentToIssue = []ActionFunc{
	RequireHTTPPost,
	RequireMattermostUserId,
	RequireInstance,
	RequireJiraUser,
	RequireJiraClient,
	handleAPIAttachCommentToIssue,
}

func handleAPIAttachCommentToIssue(a *Action) error {
	api := a.API

	attach := &struct {
		PostId   string `json:"post_id"`
		IssueKey string `json:"issueKey"`
	}{}
	err := json.NewDecoder(a.HTTPRequest.Body).Decode(&attach)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err,
			"failed to decode incoming request")
	}

	// Add a permalink to the post to the issue description
	post, appErr := api.GetPost(attach.PostId)
	if appErr != nil || post == nil {
		a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load or find post %q", attach.PostId)
	}

	commentUser, appErr := api.GetUser(post.UserId)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load User %q", post.UserId)
	}

	permalink, err := getPermaLink(a, attach.PostId, post)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to get permalink for %q", attach.PostId)
	}

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message|%s] *from @%s*\n",
		a.JiraUser.User.Name, permalink, commentUser.Username)

	var jiraComment jira.Comment
	jiraComment.Body = permalinkMessage
	jiraComment.Body += post.Message

	commentAdded, _, err := a.JiraClient.Issue.AddComment(attach.IssueKey, &jiraComment)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to attach the comment, postId: %q", attach.PostId)
	}

	rootId := attach.PostId
	parentId := ""
	if post.ParentId != "" {
		// the original post was a reply
		rootId = post.RootId
		parentId = attach.PostId
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message: fmt.Sprintf("Message attached to [%v](%v/browse/%v)",
			attach.IssueKey, a.Instance.GetURL(), attach.IssueKey),
		ChannelId: post.ChannelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    a.MattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to create notification post %q", attach.PostId)
	}

	return a.RespondJSON(commentAdded)
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

func getPermaLink(a *Action, postId string, post *model.Post) (string, error) {
	channel, appErr := a.API.GetChannel(post.ChannelId)
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to get ChannelId, ChannelId: "+post.ChannelId)
	}

	team, appErr := a.API.GetTeam(channel.TeamId)
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to get team, TeamId: "+channel.TeamId)
	}

	permalink := fmt.Sprintf("%v/%v/pl/%v",
		a.PluginConfig.SiteURL,
		team.Name,
		postId,
	)
	return permalink, nil
}

func transitionJiraIssue(a *Action, issueKey, toState string) (string, error) {
	transitions, _, err := a.JiraClient.Issue.GetTransitions(issueKey)
	if err != nil {
		return "", errors.New("We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue.")
	}
	if len(transitions) < 1 {
		return "", errors.New("You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.")
	}

	transitionToUse := jira.Transition{}
	matchingStates := []string{}
	availableStates := []string{}
	for _, transition := range transitions {
		if strings.Contains(strings.ToLower(transition.To.Name), strings.ToLower(toState)) {
			matchingStates = append(matchingStates, transition.To.Name)
			transitionToUse = transition
		}
		availableStates = append(availableStates, transition.To.Name)
	}

	switch len(matchingStates) {
	case 0:
		return "", errors.Errorf("%q is not a valid state. Please use one of: %q",
			toState, strings.Join(availableStates, ", "))

	case 1:
		// proceed

	default:
		return "", errors.Errorf("please be more specific, %q matched several states: %q",
			toState, strings.Join(matchingStates, ", "))
	}

	if _, err := a.JiraClient.Issue.DoTransition(issueKey, transitionToUse.ID); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("[%s](%v/browse/%v) transitioned to `%s`", issueKey, a.Instance.GetURL(), issueKey, transitionToUse.To.Name)
	return msg, nil
}
