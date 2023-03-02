// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	labelsField      = "labels"
	statusField      = "status"
	reporterField    = "reporter"
	priorityField    = "priority"
	descriptionField = "description"
	resolutionField  = "resolution"
)

func makePost(userID, channelID, message string) *model.Post {
	return &model.Post{
		UserId:    userID,
		ChannelId: channelID,
		Message:   message,
	}
}

func (p *Plugin) httpShareIssuePublicly(w http.ResponseWriter, r *http.Request) (int, error) {
	var requestData model.PostActionIntegrationRequest
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.Wrap(err, "unmarshall the body"))
	}

	jiraBotID := p.getUserID()
	channelID := requestData.ChannelId
	mattermostUserID := requestData.UserId
	if mattermostUserID == "" {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"user not authorized"), w, http.StatusUnauthorized)
	}

	val := requestData.Context["issue_key"]
	issueKey, ok := val.(string)
	if !ok {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"No issue key was found in context data"), w, http.StatusInternalServerError)
	}

	val = requestData.Context["instance_id"]
	instanceID, ok := val.(string)
	if !ok {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"No instance id was found in context data"), w, http.StatusInternalServerError)
	}

	_, instance, connection, err := p.getClient(types.ID(instanceID), types.ID(mattermostUserID))
	if err != nil {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"No connection could be loaded with given params"), w, http.StatusInternalServerError)
	}

	attachment, err := p.getIssueAsSlackAttachment(instance, connection, strings.ToUpper(issueKey), false)
	if err != nil {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"Could not get issue as slack attachment"), w, http.StatusInternalServerError)
	}

	post := &model.Post{
		UserId:    mattermostUserID,
		ChannelId: channelID,
	}
	post.AddProp("attachments", attachment)

	_, appErr := p.API.CreatePost(post)
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post"))
	}

	p.API.DeleteEphemeralPost(mattermostUserID, requestData.PostId)

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write([]byte(`{statusField: "OK"}`))
	return http.StatusOK, err
}

func (p *Plugin) httpTransitionIssuePostAction(w http.ResponseWriter, r *http.Request) (int, error) {
	var requestData model.PostActionIntegrationRequest
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.New("unmarshall the body"))
	}

	jiraBotID := p.getUserID()
	channelID := requestData.ChannelId

	mattermostUserID := requestData.UserId
	if mattermostUserID == "" {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"user not authorized"), w, http.StatusUnauthorized)
	}

	val := requestData.Context["issue_key"]
	issueKey, ok := val.(string)
	if !ok {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"No issue key was found in context data"), w, http.StatusInternalServerError)
	}

	val = requestData.Context["selected_option"]
	toState, ok := val.(string)
	if !ok {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"No transition option was found in context data"), w, http.StatusInternalServerError)
	}

	val = requestData.Context["instance_id"]
	instanceID, ok := val.(string)
	if !ok {
		return p.respondErrWithFeedback(mattermostUserID, makePost(jiraBotID, channelID,
			"No instance id was found in context data"), w, http.StatusInternalServerError)
	}

	_, err = p.TransitionIssue(&InTransitionIssue{
		mattermostUserID: types.ID(mattermostUserID),
		InstanceID:       types.ID(instanceID),
		IssueKey:         issueKey,
		ToState:          toState,
	})
	if err != nil {
		_ = p.API.SendEphemeralPost(mattermostUserID, makePost(jiraBotID, channelID, "Failed to transition this issue."))
		return respondErr(w, http.StatusInternalServerError, err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write([]byte(`{statusField: "OK"}`))
	return http.StatusOK, err
}

func (p *Plugin) respondErrWithFeedback(mattermostUserID string, post *model.Post, w http.ResponseWriter, status int) (int, error) {
	_ = p.API.SendEphemeralPost(mattermostUserID, post)
	return respondErr(w, status, errors.New(post.Message))
}

type InCreateIssue struct {
	mattermostUserID         types.ID
	InstanceID               types.ID         `json:"instance_id"`
	RequiredFieldsNotCovered [][]string       `json:"required_fields_not_covered"`
	PostID                   string           `json:"post_id"`
	CurrentTeam              string           `json:"current_team"`
	ChannelID                string           `json:"channel_id"`
	Fields                   jira.IssueFields `json:"fields"`
}

func (p *Plugin) httpCreateIssue(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be POST"))
	}

	in := InCreateIssue{}
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	in.mattermostUserID = types.ID(r.Header.Get("Mattermost-User-Id"))
	if in.mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	created, err := p.CreateIssue(&in)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	return respondJSON(w, created)
}

func (p *Plugin) CreateIssue(in *InCreateIssue) (*jira.Issue, error) {
	client, instance, connection, err := p.getClient(in.InstanceID, in.mattermostUserID)
	if err != nil {
		return nil, err
	}

	var post *model.Post
	var appErr *model.AppError

	// If this issue is attached to a post, lets add a permalink to the post in the Jira Description
	if in.PostID != "" {
		post, appErr = p.API.GetPost(in.PostID)
		if appErr != nil {
			return nil, errors.WithMessage(appErr, "failed to load post "+in.PostID)
		}
		if post == nil {
			return nil, errors.New("failed to load post " + in.PostID + ": not found")
		}
		permalink := getPermaLink(instance, in.PostID, in.CurrentTeam)

		if len(in.Fields.Description) > 0 {
			in.Fields.Description += fmt.Sprintf("\n\n_Issue created from a [message in Mattermost|%v]_.", permalink)
		} else {
			in.Fields.Description = fmt.Sprintf("_Issue created from a [message in Mattermost|%v]_.", permalink)
		}
	}

	rootID := in.PostID
	if post != nil && post.RootId != "" {
		// the original post was a reply
		rootID = post.RootId
	}

	issue := &jira.Issue{
		Fields: &in.Fields,
	}

	channelID := in.ChannelID
	if post != nil {
		channelID = post.ChannelId
	}

	for i, notCovered := range in.RequiredFieldsNotCovered {
		// First position in the slice is the key value (shouldn't change, regardless of localization)
		if strings.ToLower(notCovered[0]) == reporterField {
			requiredFieldsNotCovered := in.RequiredFieldsNotCovered[:i]
			if i+1 < len(in.RequiredFieldsNotCovered) {
				requiredFieldsNotCovered = append(requiredFieldsNotCovered,
					in.RequiredFieldsNotCovered[i+1:]...)
			}
			in.RequiredFieldsNotCovered = requiredFieldsNotCovered

			if instance.Common().Type == ServerInstanceType {
				issue.Fields.Reporter = &connection.User
			}
			break
		}
	}

	project, err := client.GetProject(issue.Fields.Project.Key)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get project %q", issue.Fields.Project.Key)
	}

	if len(in.RequiredFieldsNotCovered) > 0 {
		createURL := MakeCreateIssueURL(instance, project, issue)

		message := "The project you tried to create an issue for has **required fields** this plugin does not yet support:"

		var fieldsString string
		for _, v := range in.RequiredFieldsNotCovered {
			// Second position in the slice is the localized name of that key.
			fieldsString += fmt.Sprintf("- %+v\n", v[1])
		}

		reply := &model.Post{
			Message:   fmt.Sprintf("[Please create your Jira issue manually](%v). %v\n%v", createURL, message, fieldsString),
			ChannelId: channelID,
			RootId:    rootID,
			UserId:    instance.Common().getConfig().botUserID,
		}
		_ = p.API.SendEphemeralPost(in.mattermostUserID.String(), reply)
		return nil, errors.Errorf("issue can not be created via API: %s", message)
	}

	created, err := client.CreateIssue(issue)
	if err != nil {
		// if have an error and Jira tells us there are required fields send user
		// link to jira with fields already filled in.  Note the user will also see
		// these errors in Jira.
		// Note that RequiredFieldsNotCovered is also empty
		if strings.Contains(err.Error(), "is required.") {
			message := fmt.Sprintf("Failed to create issue. Your Jira project requires fields the plugin does not yet support. "+
				"[Please create your Jira issue manually](%s) or contact your Jira administrator.\n%v",
				MakeCreateIssueURL(instance, project, issue),
				err)

			_ = p.API.SendEphemeralPost(in.mattermostUserID.String(), &model.Post{
				Message:   message,
				ChannelId: channelID,
				RootId:    rootID,
				UserId:    instance.Common().getConfig().botUserID,
			})
			return nil, errors.Errorf("issue can not be created via API: %s", message)
		}

		return nil, errors.WithMessage(err, "failed to create issue")
	}

	// Reply with an ephemeral post with the Jira issue formatted as slack attachment.
	msg := fmt.Sprintf("Created Jira issue [%s](%s/browse/%s)", created.Key, instance.GetURL(), created.Key)

	reply := &model.Post{
		Message:   msg,
		ChannelId: channelID,
		RootId:    rootID,
		UserId:    instance.Common().getConfig().botUserID,
	}

	attachment, err := instance.Common().getIssueAsSlackAttachment(instance, connection, created.Key, true)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create notification post "+in.PostID)
	}

	reply.AddProp("attachments", attachment)
	_ = p.API.SendEphemeralPost(in.mattermostUserID.String(), reply)

	// Fetching issue details as Jira only returns the issue id and issue key at the time of
	// issue creation. We will not have issue summary in the creation response.
	createdIssue, err := client.GetIssue(created.Key, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to fetch issue details "+created.Key)
	}

	p.UpdateUserDefaults(in.mattermostUserID, in.InstanceID, project.Key)

	// Create a public post for all the channel members
	publicReply := &model.Post{
		Message:   fmt.Sprintf("Created a Jira issue: %s", mdKeySummaryLink(createdIssue)),
		ChannelId: channelID,
		RootId:    rootID,
		UserId:    in.mattermostUserID.String(),
	}
	_, appErr = p.API.CreatePost(publicReply)
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed to create notification post "+in.PostID)
	}

	if post != nil && len(post.FileIds) > 0 {
		go func() {
			conf := instance.Common().getConfig()
			for _, fileID := range post.FileIds {
				mattermostName, _, _, e := client.AddAttachment(p.API, created.ID, fileID, conf.maxAttachmentSize)
				if e != nil {
					notifyOnFailedAttachment(instance, in.mattermostUserID.String(), created.Key, e, "file: %s", mattermostName)
				}
			}
		}()
	}

	return createdIssue, nil
}

func (p *Plugin) httpGetCreateIssueMetadataForProjects(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("Request: "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	projectKeys := r.FormValue("project-keys")
	if projectKeys == "" {
		return respondErr(w, http.StatusBadRequest,
			errors.New("project-keys query param is required"))
	}

	instanceID := r.FormValue("instance_id")

	cimd, err := p.GetCreateIssueMetadataForProjects(types.ID(instanceID), types.ID(mattermostUserID), projectKeys)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	if len(cimd.Projects) == 0 {
		return respondJSON(w, map[string]interface{}{
			"error": "You do not have permission to create issues in that project. Please contact your Jira admin.",
		})
	}

	return respondJSON(w, cimd)
}

func (p *Plugin) GetCreateIssueMetadataForProjects(instanceID, mattermostUserID types.ID, projectKeys string) (*jira.CreateMetaInfo, error) {
	client, _, _, err := p.getClient(instanceID, mattermostUserID)
	if err != nil {
		return nil, err
	}

	return client.GetCreateMetaInfo(p.API, &jira.GetQueryOptions{
		Expand:      "projects.issuetypes.fields",
		ProjectKeys: projectKeys,
	})
}

func (p *Plugin) httpGetSearchIssues(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("Request: "+r.Method+" is not allowed, must be GET"))
	}
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}
	instanceID := r.FormValue("instance_id")
	q := r.FormValue("q")
	jqlString := r.FormValue("jql")
	fieldsStr := r.FormValue("fields")
	limitStr := r.FormValue("limit")

	result, err := p.GetSearchIssues(types.ID(instanceID), types.ID(mattermostUserID), q, jqlString, fieldsStr, limitStr)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	return respondJSON(w, result)
}

func (p *Plugin) GetSearchIssues(instanceID, mattermostUserID types.ID, q, jqlString, fieldsStr, limitStr string) ([]jira.Issue, error) {
	client, _, _, err := p.getClient(instanceID, mattermostUserID)
	if err != nil {
		return nil, err
	}

	if len(fieldsStr) == 0 {
		fieldsStr = "key,summary"
	}
	if len(jqlString) == 0 {
		escaped := strings.ReplaceAll(q, `"`, `\"`)
		jqlString = fmt.Sprintf(`text ~ "%s" OR text ~ "%s*"`, escaped, escaped)
	}

	limit := 50
	if len(limitStr) > 0 {
		parsedLimit, parseErr := strconv.Atoi(limitStr)
		if parseErr == nil {
			limit = parsedLimit
		}
	}

	fields := strings.Split(fieldsStr, ",")

	var exact *jira.Issue
	var wg sync.WaitGroup
	if reJiraIssueKey.MatchString(q) {
		wg.Add(1)
		go func() {
			exact, _ = client.GetIssue(q, &jira.GetQueryOptions{Fields: fieldsStr})
			wg.Done()
		}()
	}

	var found []jira.Issue
	wg.Add(1)
	go func() {
		found, _ = client.SearchIssues(jqlString, &jira.SearchOptions{
			MaxResults: limit,
			Fields:     fields,
		})

		wg.Done()
	}()

	wg.Wait()

	result := []jira.Issue{}
	if exact != nil {
		result = append(result, *exact)
	}

	result = append(result, found...)

	return result, nil
}

type OutProjectMetadata struct {
	Projects          []utils.ReactSelectOption            `json:"projects"`
	IssuesPerProjects map[string][]utils.ReactSelectOption `json:"issues_per_project"`
	DefaultProjectKey string                               `json:"default_project_key,omitempty"`
}

func (p *Plugin) httpGetJiraProjectMetadata(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("Request: "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	instanceID := r.FormValue("instance_id")

	plist, connection, err := p.ListJiraProjects(types.ID(instanceID), types.ID(mattermostUserID), true)
	if err != nil {
		// Getting the issue Types separately only when the status code returned is 400
		if !strings.Contains(err.Error(), "400") {
			return respondErr(w, http.StatusInternalServerError,
				errors.WithMessage(err, "failed to GetProjectMetadata"))
		}

		plist, connection, err = p.ListJiraProjects(types.ID(instanceID), types.ID(mattermostUserID), false)
		if err != nil {
			return respondErr(w, http.StatusInternalServerError,
				errors.WithMessage(err, "failed to get the list of Jira Projects"))
		}

		var projectList jira.ProjectList
		for _, prj := range plist {
			issueTypeList, iErr := p.GetIssueTypes(types.ID(instanceID), types.ID(mattermostUserID), prj.ID)
			if iErr != nil {
				p.API.LogDebug("Failed to get issue types for project.", "ProjectKey", prj.Key, "Error", iErr.Error())
				continue
			}
			prj.IssueTypes = issueTypeList
			projectList = append(projectList, prj)
		}
		plist = projectList
	}

	if len(plist) == 0 {
		_, err = respondJSON(w, map[string]interface{}{
			"error": "You do not have permission to create issues in any projects. Please contact your Jira admin.",
		})
		if err != nil {
			return respondErr(w, http.StatusInternalServerError,
				errors.WithMessage(err, "failed to create response"))
		}
	}

	projects := []utils.ReactSelectOption{}
	issues := map[string][]utils.ReactSelectOption{}
	for _, prj := range plist {
		projects = append(projects, utils.ReactSelectOption{
			Value: prj.Key,
			Label: prj.Name,
		})
		issueTypes := []utils.ReactSelectOption{}
		for _, issue := range prj.IssueTypes {
			if issue.Subtask {
				continue
			}
			issueTypes = append(issueTypes, utils.ReactSelectOption{
				Value: issue.ID,
				Label: issue.Name,
			})
		}
		issues[prj.Key] = issueTypes
	}

	return respondJSON(w, OutProjectMetadata{
		Projects:          projects,
		IssuesPerProjects: issues,
		DefaultProjectKey: connection.DefaultProjectKey,
	})
}

func (p *Plugin) ListJiraProjects(instanceID, mattermostUserID types.ID, expandIssueTypes bool) (jira.ProjectList, *Connection, error) {
	client, _, connection, err := p.getClient(instanceID, mattermostUserID)
	if err != nil {
		return nil, nil, err
	}
	plist, err := client.ListProjects("", -1, expandIssueTypes)
	if err != nil {
		return nil, nil, err
	}
	return plist, connection, nil
}

func (p *Plugin) GetIssueTypes(instanceID, mattermostUserID types.ID, projectID string) ([]jira.IssueType, error) {
	client, _, _, err := p.getClient(instanceID, mattermostUserID)
	if err != nil {
		return nil, err
	}

	issueTypes, err := client.GetIssueTypes(projectID)
	if err != nil {
		return nil, err
	}

	return issueTypes, nil
}

var reJiraIssueKey = regexp.MustCompile(`^([[:alnum:]]+)-([[:digit:]]+)$`)

func (p *Plugin) httpAttachCommentToIssue(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be POST"))
	}

	in := InAttachCommentToIssue{}
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	in.mattermostUserID = types.ID(r.Header.Get("Mattermost-User-Id"))
	if in.mattermostUserID == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	added, err := p.AttachCommentToIssue(&in)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to attach comment to issue"))
	}

	return respondJSON(w, added)
}

type InAttachCommentToIssue struct {
	mattermostUserID types.ID
	InstanceID       types.ID `json:"instance_id"`
	PostID           string   `json:"post_id"`
	CurrentTeam      string   `json:"current_team"`
	IssueKey         string   `json:"issueKey"`
}

func (p *Plugin) AttachCommentToIssue(in *InAttachCommentToIssue) (*jira.Comment, error) {
	client, instance, connection, err := p.getClient(in.InstanceID, in.mattermostUserID)
	if err != nil {
		return nil, err
	}

	// Lets add a permalink to the post in the Jira Description
	post, appErr := p.API.GetPost(in.PostID)
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed to load post "+in.PostID)
	}
	if post == nil {
		return nil, errors.New("failed to load post " + in.PostID + ": not found")
	}

	commentUser, appErr := p.API.GetUser(post.UserId)
	if appErr != nil {
		return nil, errors.New("failed to load post.UserID " + post.UserId + ": not found")
	}

	permalink := getPermaLink(instance, in.PostID, in.CurrentTeam)

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message|%s] *from @%s*\n", connection.DisplayName, permalink, commentUser.Username)

	jiraComment := jira.Comment{
		Body: permalinkMessage + post.Message,
	}

	added, err := client.AddComment(in.IssueKey, &jiraComment)
	if err != nil {
		if strings.Contains(err.Error(), "you do not have the permission to comment on this issue") {
			return nil, errors.New("you do not have permission to create a comment in the selected Jira issue. Please choose another issue or contact your Jira admin")
		}

		// The error was not a permissions error; it was unanticipated. Return it to the client.
		return nil, errors.WithMessage(err, "failed to attach the comment, postId: "+in.PostID)
	}

	go func() {
		conf := instance.Common().getConfig()
		extraText := ""
		for _, fileID := range post.FileIds {
			mattermostName, jiraName, mime, e := client.AddAttachment(p.API, in.IssueKey, fileID, conf.maxAttachmentSize)
			if e != nil {
				notifyOnFailedAttachment(instance, in.mattermostUserID.String(), in.IssueKey, e, "file: %s", mattermostName)
				continue
			}
			if isImageMIME(mime) || isEmbbedableMIME(mime) {
				extraText += "\n\nAttachment: !" + jiraName + "!"
			} else {
				extraText += "\n\nAttachment: [^" + jiraName + "]"
			}
		}
		if extraText == "" {
			return
		}

		jiraComment.ID = added.ID
		jiraComment.Body += extraText
		_, err = client.UpdateComment(in.IssueKey, &jiraComment)
		if err != nil {
			notifyOnFailedAttachment(instance, in.mattermostUserID.String(), in.IssueKey, err, "failed to completely update comment with attachments")
		}
	}()

	rootID := in.PostID
	if post.RootId != "" {
		// the original post was a reply
		rootID = post.RootId
	}

	p.UpdateUserDefaults(in.mattermostUserID, in.InstanceID, "")

	msg := fmt.Sprintf("Message attached to [%s](%s/browse/%s)", in.IssueKey, instance.GetURL(), in.IssueKey)

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message:   msg,
		ChannelId: post.ChannelId,
		RootId:    rootID,
		UserId:    in.mattermostUserID.String(),
	}
	_, appErr = p.API.CreatePost(reply)
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed to create notification post "+in.PostID)
	}

	return added, nil
}

func notifyOnFailedAttachment(instance Instance, mattermostUserID, issueKey string, err error, format string, args ...interface{}) {
	msg := "Failed to attach to issue: " + issueKey + ", " + fmt.Sprintf(format, args...)

	instance.Common().Plugin.API.LogError(fmt.Sprintf("%s: %v", msg, err), "issue", issueKey)
	errMsg := err.Error()
	if len(errMsg) > 2048 {
		errMsg = errMsg[:2048]
	}
	_, _ = instance.Common().Plugin.CreateBotDMtoMMUserID(mattermostUserID,
		"%s. Please notify your system administrator.\n%s", msg, errMsg)
}

func getPermaLink(instance Instance, postID string, currentTeam string) string {
	return fmt.Sprintf("%v/%v/pl/%v", instance.Common().Plugin.GetSiteURL(), currentTeam, postID)
}

func (p *Plugin) getIssueDataForCloudWebhook(instance Instance, issueKey string) (*jira.Issue, error) {
	ci, ok := instance.(*cloudInstance)
	if !ok {
		return nil, errors.Errorf("Must be a JIRA Cloud instance, is %s", instance.Common().Type)
	}

	jiraClient, err := ci.getClientForBot()
	if err != nil {
		return nil, err
	}

	issue, resp, err := jiraClient.Issue.Get(issueKey, nil)
	if err != nil {
		switch {
		case resp == nil:
			return nil, errors.WithMessage(userFriendlyJiraError(nil, err),
				"request to Jira failed")

		case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized:
			return nil, errors.New(`we couldn't find the issue key, or the cloud "bot" client does not have the appropriate permissions to view the issue`)
		}
	}

	return issue, nil
}

func getIssueCustomFieldValue(issue *jira.Issue, key string) StringSet {
	m, exists := issue.Fields.Unknowns.Value(key)
	if !exists || m == nil {
		return nil
	}

	switch value := m.(type) {
	case string:
		return NewStringSet(value)
	case []string:
		return NewStringSet(value...)
	case []interface{}:
		// multi-select value
		// Checkboxes, multi-select dropdown
		result := NewStringSet()
		for _, v := range value {
			s, ok := v.(string)
			if ok {
				result = result.Add(s)
				continue
			}

			obj, ok := v.(map[string]interface{})
			if !ok {
				return nil
			}
			id, ok := obj["id"].(string)
			if !ok {
				return nil
			}
			result = result.Add(id)
		}
		return result
	case map[string]interface{}:
		// single-select value
		// Radio buttons, single-select dropdown
		id, ok := value["id"].(string)
		if !ok {
			return nil
		}
		return NewStringSet(id)
	}

	return nil
}

func getIssueFieldValue(issue *jira.Issue, key string) StringSet {
	key = strings.ToLower(key)
	switch key {
	case statusField:
		return NewStringSet(issue.Fields.Status.ID)
	case labelsField:
		return NewStringSet(issue.Fields.Labels...)
	case priorityField:
		if issue.Fields.Priority != nil {
			return NewStringSet(issue.Fields.Priority.ID)
		}
	case "fixversions":
		result := NewStringSet()
		for _, v := range issue.Fields.FixVersions {
			result = result.Add(v.ID)
		}
		return result
	case "versions":
		result := NewStringSet()
		for _, v := range issue.Fields.AffectsVersions {
			result = result.Add(v.ID)
		}
		return result
	case "components":
		result := NewStringSet()
		for _, v := range issue.Fields.Components {
			result = result.Add(v.ID)
		}
		return result
	default:
		value := getIssueCustomFieldValue(issue, key)
		if value != nil {
			return value
		}
	}

	return NewStringSet()
}

func (p *Plugin) getIssueAsSlackAttachment(instance Instance, connection *Connection, issueKey string, showActions bool) ([]*model.SlackAttachment, error) {
	client, err := instance.GetClient(connection)
	if err != nil {
		return nil, err
	}

	issue, err := client.GetIssue(issueKey, nil)
	if err != nil {
		switch StatusCode(err) {
		case http.StatusNotFound:
			return nil, errors.New("we couldn't find the issue key, or you do not have the appropriate permissions to view the issue. Please try again or contact your Jira administrator")

		case http.StatusUnauthorized:
			return nil, errors.New("you do not have the appropriate permissions to view the issue. Please contact your Jira administrator")

		default:
			return nil, errors.WithMessage(err, "request to Jira failed")
		}
	}

	return asSlackAttachment(instance.GetID(), client, issue, showActions)
}

func (p *Plugin) UnassignIssue(instance Instance, mattermostUserID types.ID, issueKey string) (string, error) {
	connection, err := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
	if err != nil {
		return "", err
	}
	client, err := instance.GetClient(connection)
	if err != nil {
		return "", err
	}

	// check for valid issue key
	_, err = client.GetIssue(issueKey, nil)
	if err != nil {
		return "", errors.Errorf("We couldn't find the issue key `%s`. Please confirm the issue key and try again.", issueKey)
	}

	if err := client.UpdateAssignee(issueKey, &jira.User{}); err != nil {
		if StatusCode(err) == http.StatusForbidden {
			return "", errors.New("You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.")
		}
		return "", err
	}

	permalink := fmt.Sprintf("%v/browse/%v", instance.GetURL(), issueKey)

	msg := fmt.Sprintf("Unassigned Jira issue [%s](%s)", issueKey, permalink)
	return msg, nil
}

const MinUserSearchQueryLength = 3

func (p *Plugin) AssignIssue(instance Instance, mattermostUserID types.ID, issueKey, userSearch string) (string, error) {
	connection, err := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
	if err != nil {
		return "", err
	}
	client, err := instance.GetClient(connection)
	if err != nil {
		return "", err
	}

	// required minimum of three letters in assignee value
	if len(userSearch) < MinUserSearchQueryLength {
		errorMsg := fmt.Sprintf("`%s` contains less than %v characters.", userSearch, MinUserSearchQueryLength)
		return errorMsg, nil
	}

	// check for valid issue key
	_, err = client.GetIssue(issueKey, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("We couldn't find the issue key `%s`.  Please confirm the issue key and try again.", issueKey)
		return errorMsg, nil
	}

	// Get list of assignable users
	jiraUsers, err := client.SearchUsersAssignableToIssue(issueKey, userSearch, 10)
	if StatusCode(err) == 401 {
		return "You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.", nil
	}
	if err != nil {
		return "", err
	}

	// handle number of returned jira users
	if len(jiraUsers) == 0 {
		return "", fmt.Errorf("we couldn't find the assignee. Please use a Jira member and try again")
	}

	if len(jiraUsers) > 1 {
		errorMsg := fmt.Sprintf("`%s` matches %d or more users.  Please specify a unique assignee.\n", userSearch, len(jiraUsers))
		for i := range jiraUsers {
			name := jiraUsers[i].DisplayName
			extra := jiraUsers[i].Name
			if jiraUsers[i].EmailAddress != "" {
				if extra != "" {
					extra += ", "
				}
				extra += jiraUsers[i].EmailAddress
			}
			if extra != "" {
				name += " (" + extra + ")"
			}
			errorMsg += fmt.Sprintf("* %+v\n", name)
		}
		return "", fmt.Errorf(errorMsg)
	}

	// user is array of one object
	user := jiraUsers[0]

	// From Jira error: query parameters 'accountId' and 'username' are mutually exclusive.
	// Here, we must choose one and one only and nil the other user field.
	// Choosing user.AccountID over user.Name, but check if AccountId is empty.
	// For server instances, AccountID is empty
	if user.AccountID != "" {
		user.Name = ""
	}

	if err := client.UpdateAssignee(issueKey, &user); err != nil {
		return "", err
	}

	permalink := fmt.Sprintf("%v/browse/%v", instance.GetURL(), issueKey)

	msg := fmt.Sprintf("`%s` assigned to Jira issue [%s](%s)", user.DisplayName, issueKey, permalink)
	return msg, nil
}

type InTransitionIssue struct {
	mattermostUserID types.ID
	InstanceID       types.ID `json:"instance_id"`
	PostToChannelID  string   `json:"channel_id"`
	IssueKey         string   `json:"issue_key"`
	ToState          string   `json:"to_state"`
}

func (p *Plugin) TransitionIssue(in *InTransitionIssue) (string, error) {
	client, instance, _, err := p.getClient(in.InstanceID, in.mattermostUserID)
	if err != nil {
		return "", err
	}

	transitions, err := client.GetTransitions(in.IssueKey)
	if err != nil {
		return "", errors.New("we couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue")
	}
	if len(transitions) < 1 {
		return "", errors.New("you do not have the appropriate permissions to perform this action. Please contact your Jira administrator")
	}

	var transition jira.Transition
	matchingStates := []string{}
	availableStates := []string{}

	potentialState := strings.ToLower(strings.Join(strings.Fields(in.ToState), ""))
	for _, t := range transitions {
		validState := strings.ToLower(strings.Join(strings.Fields(t.To.Name), ""))
		if strings.Contains(validState, potentialState) {
			matchingStates = append(matchingStates, t.To.Name)
			transition = t
		}
		availableStates = append(availableStates, t.To.Name)
	}

	switch len(matchingStates) {
	case 0:
		return "", errors.Errorf("%q is not a valid state. Please use one of: %q",
			in.ToState, strings.Join(availableStates, ", "))

	case 1:
		// proceed

	default:
		return "", errors.Errorf("please be more specific, %q matched several states: %q",
			in.ToState, strings.Join(matchingStates, ", "))
	}

	err = client.DoTransition(in.IssueKey, transition.ID)
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf("[%s](%v/browse/%v) transitioned to `%s`",
		in.IssueKey, instance.GetURL(), in.IssueKey, transition.To.Name)

	issue, err := client.GetIssue(in.IssueKey, nil)
	if err != nil {
		switch StatusCode(err) {
		case http.StatusNotFound:
			return "", errors.New("we couldn't find the issue key, or you do not have the appropriate permissions to view the issue. Please try again or contact your Jira administrator")

		case http.StatusUnauthorized:
			return "", errors.New("you do not have the appropriate permissions to view the issue. Please contact your Jira administrator")

		default:
			return "", errors.WithMessage(err, "request to Jira failed")
		}
	}

	attachments, err := asSlackAttachment(instance.GetID(), client, issue, true)
	if err != nil {
		return "", err
	}

	post := makePost(p.getUserID(), in.PostToChannelID, msg)
	post.AddProp("attachments", attachments)
	_ = p.API.SendEphemeralPost(in.mattermostUserID.String(), post)

	return msg, nil
}

func (p *Plugin) getClient(instanceID, mattermostUserID types.ID) (Client, Instance, *Connection, error) {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return nil, nil, nil, err
	}
	connection, err := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
	if err != nil {
		return nil, nil, nil, err
	}
	client, err := instance.GetClient(connection)
	if err != nil {
		return nil, nil, nil, err
	}
	return client, instance, connection, nil
}
