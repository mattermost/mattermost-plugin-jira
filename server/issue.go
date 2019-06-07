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

	jira "github.com/andygrunwald/go-jira"
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
		PostId      string           `json:"post_id"`
		CurrentTeam string           `json:"current_team"`
		ChannelId   string           `json:"channel_id"`
		Fields      jira.IssueFields `json:"fields"`
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

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
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
	if create.PostId != "" {
		post, appErr = api.GetPost(create.PostId)
		if appErr != nil {
			return http.StatusInternalServerError,
				errors.WithMessage(appErr, "failed to load post "+create.PostId)
		}
		if post == nil {
			return http.StatusInternalServerError,
				errors.New("failed to load post " + create.PostId + ": not found")
		}
		permalink := getPermaLink(ji, create.PostId, create.CurrentTeam)

		if len(create.Fields.Description) > 0 {
			create.Fields.Description += fmt.Sprintf("\n\n_Issue created from a [message in Mattermost|%v]_.", permalink)
		} else {
			create.Fields.Description = fmt.Sprintf("_Issue created from a [message in Mattermost|%v]_.", permalink)
		}
	}

	created, resp, err := jiraClient.Issue.Create(&jira.Issue{
		Fields: &create.Fields,
	})

	// For now, if we are not attaching to a post, leave postId blank (this will only affect the error message)
	postId := ""
	channelId := create.ChannelId
	if post != nil {
		channelId = post.ChannelId
		postId = create.PostId
	}

	if err != nil {
		message := "failed to create the issue, postId: " + create.PostId + ", channelId: " + channelId
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return http.StatusInternalServerError, errors.WithMessage(err, message)
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

	rootId := create.PostId
	parentId := ""
	if post.ParentId != "" {
		// the original post was a reply
		rootId = post.RootId
		parentId = create.PostId
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		// TODO: Why is this not created.Self?
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", ji.GetURL(), created.Key),
		ChannelId: channelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post, postId: "+postId+", channelId: "+channelId)
	}

	userBytes, err := json.Marshal(created)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to marshal response, postId: "+postId+", channelId: "+channelId)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(userBytes)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response, postId: "+postId+", channelId: "+channelId)
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

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

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
		return http.StatusInternalServerError,
			errors.WithMessage(err, message)
	}

	w.Header().Set("Content-Type", "application/json")

	var bb []byte
	if len(cimd.Projects) == 0 {
		bb = []byte(`{"error": "You do not have permission to create issues in any projects. Please contact your Jira admin."}`)
	} else {
		bb, err = json.Marshal(cimd)
		if err != nil {
			return http.StatusInternalServerError,
				errors.WithMessage(err, "failed to marshal response")
		}
	}
	_, err = w.Write(bb)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response")
	}

	return http.StatusOK, nil
}

func httpAPIGetSearchIssues(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("Request: " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jqlString := r.FormValue("jql")

	searchRes, resp, err := jiraClient.Issue.Search(jqlString, &jira.SearchOptions{
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
		return http.StatusInternalServerError,
			errors.WithMessage(err, message)
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

	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(resSummary)
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

func httpAPIAttachCommentToIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be POST")
	}

	api := ji.GetPlugin().API

	attach := &struct {
		PostId      string `json:"post_id"`
		CurrentTeam string `json:"current_team"`
		IssueKey    string `json:"issueKey"`
	}{}
	err := json.NewDecoder(r.Body).Decode(&attach)
	if err != nil {
		return http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Lets add a permalink to the post in the Jira Description
	post, appErr := api.GetPost(attach.PostId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load post "+attach.PostId)
	}
	if post == nil {
		return http.StatusInternalServerError,
			errors.New("failed to load post " + attach.PostId + ": not found")
	}

	commentUser, appErr := api.GetUser(post.UserId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.New("failed to load post.UserID " + post.UserId + ": not found")
	}

	permalink := getPermaLink(ji, attach.PostId, attach.CurrentTeam)

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message|%s] *from @%s*\n", jiraUser.User.Name, permalink, commentUser.Username)

	var jiraComment jira.Comment
	jiraComment.Body = permalinkMessage
	jiraComment.Body += post.Message

	commentAdded, _, err := jiraClient.Issue.AddComment(attach.IssueKey, &jiraComment)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to attach the comment, postId: "+attach.PostId)
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
		Message:   fmt.Sprintf("Message attached to [%v](%v/browse/%v)", attach.IssueKey, ji.GetURL(), attach.IssueKey),
		ChannelId: post.ChannelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post "+attach.PostId)
	}

	userBytes, err := json.Marshal(commentAdded)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to marshal response "+attach.PostId)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(userBytes)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response "+attach.PostId)
	}
	return http.StatusOK, nil
}

func getPermaLink(ji Instance, postId string, currentTeam string) string {
	return fmt.Sprintf("%v/%v/pl/%v", ji.GetPlugin().GetSiteURL(), currentTeam, postId)
}

func (p *Plugin) getIssueAsSlackAttachment(ji Instance, jiraUser JIRAUser, issueKey string) ([]*model.SlackAttachment, error) {
	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return nil, err
	}

	issue, resp, err := jiraClient.Issue.Get(issueKey, nil)
	if err != nil {
		message := "request to Jira failed"
		if resp != nil {
			if resp.StatusCode == http.StatusNotFound {
				return nil, errors.New("We couldn't find the issue key, or you do not have the appropriate permissions to view the issue. Please try again or contact your Jira administrator.")
			}
			if resp.StatusCode == http.StatusUnauthorized {
				return nil, errors.New("You do not have the appropriate permissions to view the issue. Please contact your Jira administrator.")
			}

			// return more detail for an exceptional error case
			bb, _ := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			message += ", details: " + string(bb)
		}
		return nil, errors.Wrap(err, message)
	}

	return parseIssue(issue), nil
}

func (p *Plugin) transitionJiraIssue(mmUserId, issueKey, toState string) (string, error) {
	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return "", err
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mmUserId)
	if err != nil {
		return "", err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return "", err
	}

	transitions, _, err := jiraClient.Issue.GetTransitions(issueKey)
	if err != nil {
		return "", errors.New("We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue.")
	}

	if len(transitions) < 1 {
		return "", errors.New("You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.")
	}

	var transitionToUse *jira.Transition
	for _, transition := range transitions {
		if strings.Contains(strings.ToLower(transition.To.Name), strings.ToLower(toState)) {
			transitionToUse = &transition
			break
		}
	}

	if transitionToUse == nil {
		return "", errors.New("We couldn't find the state. Please use a Jira state such as 'done' and try again.")
	}

	if _, err := jiraClient.Issue.DoTransition(issueKey, transitionToUse.ID); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("[%s](%v/browse/%v) transitioned to `%s`", issueKey, ji.GetURL(), issueKey, toState)
	return msg, nil
}
