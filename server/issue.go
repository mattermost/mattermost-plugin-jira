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
		RequiredFieldsNotCovered [][]string       `json:"required_fields_not_covered"`
		PostId                   string           `json:"post_id"`
		CurrentTeam              string           `json:"current_team"`
		ChannelId                string           `json:"channel_id"`
		Fields                   jira.IssueFields `json:"fields"`
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

	rootId := create.PostId
	parentId := ""
	if post.ParentId != "" {
		// the original post was a reply
		rootId = post.RootId
		parentId = create.PostId
	}

	issue := &jira.Issue{
		Fields: &create.Fields,
	}

	channelId := create.ChannelId
	if post != nil {
		channelId = post.ChannelId
	}

	for i, notCovered := range create.RequiredFieldsNotCovered {
		// First position in the slice is the key value (shouldn't change, regardless of localization)
		if strings.ToLower(notCovered[0]) == "reporter" {
			requiredFieldsNotCovered := create.RequiredFieldsNotCovered[:i]
			if i+1 < len(create.RequiredFieldsNotCovered) {
				requiredFieldsNotCovered = append(requiredFieldsNotCovered,
					create.RequiredFieldsNotCovered[i+1:]...)
			}
			create.RequiredFieldsNotCovered = requiredFieldsNotCovered

			if ji.GetType() == JIRATypeServer {
				issue.Fields.Reporter = &jiraUser.User
			}
			break
		}
	}

	project, resp, err := jiraClient.Project.Get(issue.Fields.Project.Key)
	if err != nil {
		message := "failed to get the project, postId: " + create.PostId
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}

		return http.StatusInternalServerError, errors.WithMessage(err, message)
	}

	if len(create.RequiredFieldsNotCovered) > 0 {

		req := buildCreateQuery(ji, project, issue)

		message := "The project you tried to create an issue for has **required fields** this plugin does not yet support:"

		var fieldsString string
		for _, v := range create.RequiredFieldsNotCovered {
			// Second position in the slice is the localized name of that key.
			fieldsString = fieldsString + fmt.Sprintf("- %+v\n", v[1])
		}

		reply := &model.Post{
			Message:   fmt.Sprintf("[Please create your Jira issue manually](%v). %v\n%v", req.URL.String(), message, fieldsString),
			ChannelId: post.ChannelId,
			RootId:    rootId,
			ParentId:  parentId,
			UserId:    ji.GetPlugin().getConfig().botUserID,
		}
		_ = api.SendEphemeralPost(mattermostUserId, reply)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{}")
		return http.StatusOK, nil
	}

	created, resp, err := jiraClient.Issue.Create(issue)
	if err != nil {
		message := "failed to create the issue, postId: " + create.PostId + ", channelId: " + channelId
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}

		// if have an error and Jira tells us there are required fields send user
		// link to jira with fields already filled in.  Note the user will also see
		// these errors in Jira.
		// Note that RequiredFieldsNotCovered is also empty
		if strings.Contains(message, "is required.") {
			req := buildCreateQuery(ji, project, issue)

			message = "This plugin did not receive all the required fields from your Jira project and could not complete the request. "
			reply := &model.Post{
				Message:   fmt.Sprintf("%v [Please create your Jira issue manually](%v) or contact your Jira administrator.", message, req.URL.String()),
				ChannelId: post.ChannelId,
				RootId:    rootId,
				ParentId:  parentId,
				UserId:    ji.GetPlugin().getConfig().botUserID,
			}

			_ = api.SendEphemeralPost(mattermostUserId, reply)

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "{}")
			return http.StatusOK, nil
		}

		// The error was not a fields required error; it was unanticipated. Return it to the client.
		return http.StatusInternalServerError,
			errors.WithMessage(err, message)
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		// TODO: Why is this not created.Self?
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", ji.GetURL(), created.Key),
		ChannelId: post.ChannelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post "+create.PostId)
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

	userBytes, err := json.Marshal(created)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to marshal response, postId: "+create.PostId+", channelId: "+channelId)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(userBytes)
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to write response, postId: "+create.PostId+", channelId: "+channelId)
	}
	return http.StatusOK, nil
}

func httpAPIGetCreateIssueMetadataForProject(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("Request: " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	projectKey := r.FormValue("project-key")
	if projectKey == "" {
		return http.StatusBadRequest, errors.New("project-key query param is required")
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
		Expand:      "projects.issuetypes.fields",
		ProjectKeys: projectKey,
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
		bb = []byte(`{"error": "You do not have permission to create issues in that project. Please contact your Jira admin."}`)
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

func httpAPIGetJiraProjectMetadata(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
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

	cimd, resp, err := jiraClient.Issue.GetCreateMetaWithOptions(nil)
	if err != nil {
		message := "failed to get CreateIssue metadata"
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return http.StatusInternalServerError, errors.WithMessage(err, message)
	}

	w.Header().Set("Content-Type", "application/json")

	type issueType struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	type project struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	type projectMetadata struct {
		Projects          []project              `json:"projects"`
		IssuesPerProjects map[string][]issueType `json:"issues_per_project"`
	}

	var bb []byte
	if len(cimd.Projects) == 0 {
		bb = []byte(`{"error": "You do not have permission to create issues in any projects. Please contact your Jira admin."}`)
	} else {
		projects := make([]project, 0, len(cimd.Projects))
		issues := make(map[string][]issueType, len(cimd.Projects))
		for _, prj := range cimd.Projects {
			projects = append(projects, project{
				Value: prj.Key,
				Label: prj.Name,
			})
			issueTypes := make([]issueType, 0, len(prj.IssueTypes))
			for _, issue := range prj.IssueTypes {
				if issue.Subtasks {
					continue
				}
				issueTypes = append(issueTypes, issueType{
					Value: issue.Id,
					Label: issue.Name,
				})
			}
			issues[prj.Key] = issueTypes
		}
		payload := projectMetadata{
			Projects:          projects,
			IssuesPerProjects: issues,
		}

		bb, err = json.Marshal(payload)
		if err != nil {
			return http.StatusInternalServerError, errors.WithMessage(err, "failed to marshal response")
		}
	}

	_, err = w.Write(bb)
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
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
		if strings.Contains(err.Error(), "you do not have the permission to comment on this issue") {
			return http.StatusNotFound,
				errors.New("You do not have permission to create a comment in the selected Jira issue. Please choose another issue or contact your Jira admin.")
		}

		// The error was not a permissions error; it was unanticipated. Return it to the client.
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

func buildCreateQuery(ji Instance, project *jira.Project, issue *jira.Issue) *http.Request {

	url := fmt.Sprintf("%v/secure/CreateIssueDetails!init.jspa", ji.GetURL())
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("we've found the errro = %+v\n", err)
	}

	q := req.URL.Query()
	q.Add("pid", project.ID)
	q.Add("issuetype", issue.Fields.Type.ID)
	q.Add("summary", issue.Fields.Summary)
	q.Add("description", issue.Fields.Description)

	// if no priority, ID field does not exist
	if issue.Fields.Priority != nil {
		q.Add("priority", issue.Fields.Priority.ID)
	}

	// add custom fields
	for k, v := range issue.Fields.Unknowns {

		strV, ok := v.(string)
		if ok {
			q.Add(k, strV)
		}

		if mapV, ok := v.(map[string]interface{}); ok {
			if id, ok := mapV["id"].(string); ok {
				q.Add(k, id)
			}
		}

	}

	req.URL.RawQuery = q.Encode()

	return req
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
		p.errorf("transitionJiraIssue: failed to load current Jira instance: %v", err)
		return "", errors.New("Failed to load current Jira instance. Please contact your system administrator.")
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
