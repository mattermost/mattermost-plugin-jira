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

func httpAPICreateIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be POST")
	}

	api := ji.GetPlugin().API

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

	jiraUser, err := ji.GetPlugin().LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Lets add a permalink to the post in the Jira Description
	post, appErr := api.GetPost(create.PostId)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load post "+create.PostId)
	}
	if post == nil {
		return http.StatusInternalServerError,
			errors.New("failed to load post " + create.PostId + ": not found")
	}

	permalink, err := getPermaLink(ji, create.PostId, post)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New("failed to get permalink for " + create.PostId + ": not found")
	}

	if len(create.Fields.Description) > 0 {
		create.Fields.Description += fmt.Sprintf("\n%v", permalink)
	} else {
		create.Fields.Description = permalink
	}

	created, resp, err := jiraClient.Issue.Create(&jira.Issue{
		Fields: &create.Fields,
	})
	if err != nil {
		message := "failed to create the issue, postId: " + create.PostId
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return http.StatusInternalServerError, errors.WithMessage(err, message)
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
		ChannelId: post.ChannelId,
		RootId:    create.PostId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
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

func httpAPIAttachCommentToIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be POST")
	}

	api := ji.GetPlugin().API

	attach := &struct {
		PostId   string `json:"post_id"`
		IssueKey string `json:"issueKey"`
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

	jiraUser, err := ji.GetPlugin().LoadJIRAUser(ji, mattermostUserId)
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

	permalink, err := getPermaLink(ji, attach.PostId, post)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New("failed to get permalink for " + attach.PostId + ": not found")
	}

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message|%s] *from @%s*\n", jiraUser.User.Name, permalink, commentUser.Username)

	var jiraComment jira.Comment
	jiraComment.Body = permalinkMessage
	jiraComment.Body += post.Message

	commentAdded, _, err := jiraClient.Issue.AddComment(attach.IssueKey, &jiraComment)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to attach the comment, postId: "+attach.PostId)
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message:   fmt.Sprintf("Message attached to [%v](%v/browse/%v)", attach.IssueKey, ji.GetURL(), attach.IssueKey),
		ChannelId: post.ChannelId,
		RootId:    attach.PostId,
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

func getPermaLink(ji Instance, postId string, post *model.Post) (string, error) {

	api := ji.GetPlugin().API

	channel, appErr := api.GetChannel(post.ChannelId)
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to get ChannelId, ChannelId: "+post.ChannelId)
	}

	team, appErr := api.GetTeam(channel.TeamId)
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to get team, TeamId: "+channel.TeamId)
	}

	permalink := fmt.Sprintf("%v/%v/pl/%v",
		ji.GetPlugin().GetSiteURL(),
		team.Name,
		postId,
	)
	return permalink, nil
}

func (p *Plugin) assignJiraIssue(mmUserId, issueKey, assignee string) (string, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return "", err
	}

	jiraUser, err := ji.GetPlugin().LoadJIRAUser(ji, mmUserId)
	if err != nil {
		return "", err
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return "", err
	}

	// check for valid issue key
	_, _, err = jiraClient.Issue.Get(issueKey, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("We couldn't find the issue key `%s`.  Please confirm the issue key and try again.", issueKey)
		return errorMsg, nil
	}

	// Get list of assignable assignees
	url := fmt.Sprintf("rest/api/3/user/assignable/search?issueKey=%s&query=%s", issueKey, assignee)
	req, err := jiraClient.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	var jiraUsers []*jira.User
	resp, err := jiraClient.Do(req, &jiraUsers)
	if err != nil {
		if resp.Response.StatusCode == 401 {
			return "You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.", nil
		}
		return "", err
	}

	// handle number of returned jira users
	if len(jiraUsers) == 0 {
		errorMsg := fmt.Sprintf("We couldn't find the assignee. Please use a Jira member and try again.")
		return "", fmt.Errorf(errorMsg)
	}

	if len(jiraUsers) > 1 {
		errorMsg := fmt.Sprintf("Your <assignee>, `%s`, matches %d users.  Please make your user request unique.", assignee, len(jiraUsers))
		return "", fmt.Errorf(errorMsg)
	}

	// user is array of one object
	user := jiraUsers[0]

	if _, err := jiraClient.Issue.UpdateAssignee(issueKey, user); err != nil {
		return "", err
	}

	permalink := fmt.Sprintf("%v/browse/%v", ji.GetURL(), issueKey)

	msg := fmt.Sprintf("`%s` assigned to Jira issue [%s](%s)", user.DisplayName, issueKey, permalink)
	return msg, nil

}

func (p *Plugin) transitionJiraIssue(mmUserId, issueKey, toState string) (string, error) {
	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return "", err
	}

	jiraUser, err := ji.GetPlugin().LoadJIRAUser(ji, mmUserId)
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
