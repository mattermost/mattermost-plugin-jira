// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-workflow-client/workflowclient"
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

func makePost(userId, channelId, message string) *model.Post {
	return &model.Post{
		UserId:    userId,
		ChannelId: channelId,
		Message:   message,
	}
}

func httpAPITransitionIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	requestData := model.PostActionIntegrationRequestFromJson(r.Body)
	if requestData == nil {
		return respondErr(w, http.StatusBadRequest,
			errors.New("Missing request data"))
	}

	plugin := ji.GetPlugin()
	jiraBotID := plugin.getUserID()
	channelID := requestData.ChannelId

	var msg string
	mattermostUserId := requestData.UserId
	if mattermostUserId == "" {
		msg = "user not authorized"
		_ = plugin.API.SendEphemeralPost(mattermostUserId, makePost(jiraBotID, channelID, msg))
		return respondErr(w, http.StatusUnauthorized, errors.New(msg))
	}

	val := requestData.Context["issueKey"]
	issueKey, ok := val.(string)
	if !ok {
		msg = "No issue key was found in context data"
		_ = plugin.API.SendEphemeralPost(mattermostUserId, makePost(jiraBotID, channelID, msg))
		return respondErr(w, http.StatusInternalServerError, errors.New(msg))
	}

	val = requestData.Context["selected_option"]
	toState, ok := val.(string)
	if !ok {
		msg = "No transition option was found in context data"
		_ = plugin.API.SendEphemeralPost(mattermostUserId, makePost(jiraBotID, channelID, msg))
		return respondErr(w, http.StatusInternalServerError, errors.New(msg))
	}

	jiraUser, err := plugin.userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		msg = "Your username is not connected to Jira. Please type `/jira connect`."
		_ = plugin.API.SendEphemeralPost(mattermostUserId, makePost(jiraBotID, channelID, msg))
		return respondErr(w, http.StatusUnauthorized, err)
	}

	msg, err = plugin.transitionJiraIssue(mattermostUserId, issueKey, toState)
	if err != nil {
		msg = "Failed to transition this issue."
		_ = plugin.API.SendEphemeralPost(mattermostUserId, makePost(jiraBotID, channelID, msg))
		return respondErr(w, http.StatusInternalServerError, err)
	}

	attachment, err := plugin.getIssueAsSlackAttachment(ji, jiraUser, issueKey)
	if err != nil {
		_ = plugin.API.SendEphemeralPost(mattermostUserId, makePost(jiraBotID, channelID, err.Error()))
		return respondErr(w, http.StatusInternalServerError, err)
	}

	post := makePost(jiraBotID, channelID, msg)
	post.AddProp("attachments", attachment)
	_ = plugin.API.SendEphemeralPost(mattermostUserId, post)

	return http.StatusOK, nil
}

func httpAPICreateIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be POST"))
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
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	var post *model.Post
	var appErr *model.AppError

	// If this issue is attached to a post, lets add a permalink to the post in the Jira Description
	if create.PostId != "" {
		post, appErr = api.GetPost(create.PostId)
		if appErr != nil {
			return respondErr(w, http.StatusInternalServerError,
				errors.WithMessage(appErr, "failed to load post "+create.PostId))
		}
		if post == nil {
			return respondErr(w, http.StatusInternalServerError,
				errors.New("failed to load post "+create.PostId+": not found"))
		}
		permalink := getPermaLink(ji, create.PostId, create.CurrentTeam)

		if len(create.Fields.Description) > 0 {
			create.Fields.Description += fmt.Sprintf("\n\n_Issue created from a [message in Mattermost|%v]_.", permalink)
		} else {
			create.Fields.Description = fmt.Sprintf("_Issue created from a [message in Mattermost|%v]_.", permalink)
		}
	}

	rootId := create.PostId
	if post != nil && post.RootId != "" {
		// the original post was a reply
		rootId = post.RootId
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

	project, err := client.GetProject(issue.Fields.Project.Key)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessagef(err, "failed to get project %q", issue.Fields.Project.Key))
	}

	if len(create.RequiredFieldsNotCovered) > 0 {
		createURL := MakeCreateIssueURL(ji, project, issue)

		message := "The project you tried to create an issue for has **required fields** this plugin does not yet support:"

		var fieldsString string
		for _, v := range create.RequiredFieldsNotCovered {
			// Second position in the slice is the localized name of that key.
			fieldsString = fieldsString + fmt.Sprintf("- %+v\n", v[1])
		}

		reply := &model.Post{
			Message:   fmt.Sprintf("[Please create your Jira issue manually](%v). %v\n%v", createURL, message, fieldsString),
			ChannelId: channelId,
			RootId:    rootId,
			ParentId:  rootId,
			UserId:    ji.GetPlugin().getConfig().botUserID,
		}
		_ = api.SendEphemeralPost(mattermostUserId, reply)
		return respondJSON(w, struct{}{})
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
				MakeCreateIssueURL(ji, project, issue),
				err)

			_ = api.SendEphemeralPost(mattermostUserId, &model.Post{
				Message:   message,
				ChannelId: channelId,
				RootId:    rootId,
				ParentId:  rootId,
				UserId:    ji.GetPlugin().getConfig().botUserID,
			})
			return respondJSON(w, struct{}{})
		}

		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to create issue"))
	}

	// Reply with an ephemeral post with the Jira issue formatted as slack attachment.
	msg := fmt.Sprintf("Created Jira issue [%s](%s/browse/%s)", created.Key, ji.GetURL(), created.Key)

	reply := &model.Post{
		Message:   msg,
		ChannelId: channelId,
		RootId:    rootId,
		ParentId:  rootId,
		UserId:    ji.GetPlugin().getConfig().botUserID,
	}

	attachment, err := ji.GetPlugin().getIssueAsSlackAttachment(ji, jiraUser, created.Key)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to create notification post "+create.PostId))
	}

	reply.AddProp("attachments", attachment)
	_ = api.SendEphemeralPost(mattermostUserId, reply)

	// Fetching issue details as Jira only returns the issue id and issue key at the time of
	// issue creation. We will not have issue summary in the creation response.
	createdIssue, err := client.GetIssue(created.Key, nil)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to fetch issue details "+created.Key))
	}

	// Create a public post for all the channel members
	publicReply := &model.Post{
		Message:   fmt.Sprintf("Created a Jira issue: %s", mdKeySummaryLink(createdIssue)),
		ChannelId: channelId,
		RootId:    rootId,
		ParentId:  rootId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(publicReply)
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post "+create.PostId))
	}

	if post != nil && len(post.FileIds) > 0 {
		go func() {
			conf := ji.GetPlugin().getConfig()
			for _, fileId := range post.FileIds {
				mattermostName, _, _, e := client.AddAttachment(api, created.ID, fileId, conf.maxAttachmentSize)
				if e != nil {
					notifyOnFailedAttachment(ji, mattermostUserId, created.Key, e, "file: %s", mattermostName)
				}
			}
		}()
	}

	return respondJSON(w, created)
}

func httpWorkflowCreateIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be POST"))
	}

	var activationParams workflowclient.ActionActivationParams
	if err := json.NewDecoder(r.Body).Decode(&activationParams); err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	create := &struct {
		UserId string           `json:"user_id"`
		Fields jira.IssueFields `json:"fields"`
	}{}
	if err := json.NewDecoder(bytes.NewReader(activationParams.Action)).Decode(&create); err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request specific parameters"))
	}

	var client interface {
		CreateIssue(issue *jira.Issue) (*jira.Issue, error)
	}
	if create.UserId != "" {
		jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, create.UserId)
		if err != nil {
			return respondErr(w, http.StatusInternalServerError, err)
		}

		client, err = ji.GetClient(jiraUser)
		if err != nil {
			return respondErr(w, http.StatusInternalServerError, err)
		}
	} else {
		jci, ok := ji.(*jiraCloudInstance)
		if !ok {
			return respondErr(w, http.StatusBadRequest,
				errors.New("UserId is required for jira server instances."))
		}

		jiraClient, err := jci.getJIRAClientForBot()
		if err != nil {
			return respondErr(w, http.StatusInternalServerError,
				fmt.Errorf("unable to get jira client for server: %w", err))
		}

		client = &JiraClient{Jira: jiraClient}
	}

	issue := &jira.Issue{
		Fields: &create.Fields,
	}

	created, err := client.CreateIssue(issue)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.Errorf("Failed to create issue. %s", err.Error()))
	}

	outputVars := make(workflowclient.Vars)
	outputVars["TicketKey"] = created.Key

	return respondJSON(w, outputVars)
}

func httpAPIGetCreateIssueMetadataForProjects(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("Request: "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	projectKeys := r.FormValue("project-keys")
	if projectKeys == "" {
		return respondErr(w, http.StatusBadRequest,
			errors.New("project-keys query param is required"))
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	cimd, err := client.GetCreateMeta(&jira.GetQueryOptions{
		Expand:      "projects.issuetypes.fields",
		ProjectKeys: projectKeys,
	})
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to GetCreateIssueMetadata"))
	}

	if len(cimd.Projects) == 0 {
		return respondJSON(w, map[string]interface{}{
			"error": "You do not have permission to create issues in that project. Please contact your Jira admin.",
		})
	}

	return respondJSON(w, cimd)
}

func httpAPIGetSearchIssues(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("Request: "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	q := r.FormValue("q")
	jqlString := r.FormValue("jql")
	fieldsStr := r.FormValue("fields")
	limitStr := r.FormValue("limit")

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

	for _, issue := range found {
		result = append(result, issue)
	}

	return respondJSON(w, result)
}

func httpAPIGetJiraProjectMetadata(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("Request: "+r.Method+" is not allowed, must be GET"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized, errors.New("not authorized"))
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	cimd, err := client.GetCreateMeta(nil)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to GetCreateIssueMetadata"))
	}

	if len(cimd.Projects) == 0 {
		respondJSON(w, map[string]interface{}{
			"error": "You do not have permission to create issues in any projects. Please contact your Jira admin.",
		})
	}

	type option = utils.ReactSelectOption
	type projectMetadata struct {
		Projects          []option            `json:"projects"`
		IssuesPerProjects map[string][]option `json:"issues_per_project"`
	}

	projects := make([]option, 0, len(cimd.Projects))
	issues := make(map[string][]option, len(cimd.Projects))
	for _, prj := range cimd.Projects {
		projects = append(projects, option{
			Value: prj.Key,
			Label: prj.Name,
		})
		issueTypes := make([]option, 0, len(prj.IssueTypes))
		for _, issue := range prj.IssueTypes {
			if issue.Subtasks {
				continue
			}
			issueTypes = append(issueTypes, option{
				Value: issue.Id,
				Label: issue.Name,
			})
		}
		issues[prj.Key] = issueTypes
	}

	return respondJSON(w, projectMetadata{
		Projects:          projects,
		IssuesPerProjects: issues,
	})
}

var reJiraIssueKey = regexp.MustCompile(`^([[:alpha:]]+)-([[:digit:]]+)$`)

func httpAPIAttachCommentToIssue(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be POST"))
	}

	api := ji.GetPlugin().API

	attach := &struct {
		PostId      string `json:"post_id"`
		CurrentTeam string `json:"current_team"`
		IssueKey    string `json:"issueKey"`
	}{}
	err := json.NewDecoder(r.Body).Decode(&attach)
	if err != nil {
		return respondErr(w, http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request"))
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return respondErr(w, http.StatusUnauthorized,
			errors.New("not authorized"))
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	// Lets add a permalink to the post in the Jira Description
	post, appErr := api.GetPost(attach.PostId)
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to load post "+attach.PostId))
	}
	if post == nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.New("failed to load post "+attach.PostId+": not found"))
	}

	commentUser, appErr := api.GetUser(post.UserId)
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.New("failed to load post.UserID "+post.UserId+": not found"))
	}

	permalink := getPermaLink(ji, attach.PostId, attach.CurrentTeam)

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message|%s] *from @%s*\n", jiraUser.User.DisplayName, permalink, commentUser.Username)

	var jiraComment jira.Comment
	jiraComment.Body = permalinkMessage
	jiraComment.Body += post.Message

	commentAdded, err := client.AddComment(attach.IssueKey, &jiraComment)
	if err != nil {
		if strings.Contains(err.Error(), "you do not have the permission to comment on this issue") {
			return respondErr(w, http.StatusNotFound,
				errors.New("You do not have permission to create a comment in the selected Jira issue. Please choose another issue or contact your Jira admin."))
		}

		// The error was not a permissions error; it was unanticipated. Return it to the client.
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(err, "failed to attach the comment, postId: "+attach.PostId))
	}

	go func() {
		conf := ji.GetPlugin().getConfig()
		extraText := ""
		for _, fileId := range post.FileIds {
			mattermostName, jiraName, mime, e := client.AddAttachment(api, attach.IssueKey, fileId, conf.maxAttachmentSize)
			if e != nil {
				notifyOnFailedAttachment(ji, mattermostUserId, attach.IssueKey, e, "file: %s", mattermostName)
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

		jiraComment.ID = commentAdded.ID
		jiraComment.Body += extraText
		_, err = client.UpdateComment(attach.IssueKey, &jiraComment)
		if err != nil {
			notifyOnFailedAttachment(ji, mattermostUserId, attach.IssueKey, err, "failed to completely update comment with attachments")
		}
	}()

	rootId := attach.PostId
	if post.RootId != "" {
		// the original post was a reply
		rootId = post.RootId
	}

	msg := fmt.Sprintf("Message attached to [%s](%s/browse/%s)", attach.IssueKey, ji.GetURL(), attach.IssueKey)

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message:   msg,
		ChannelId: post.ChannelId,
		RootId:    rootId,
		ParentId:  rootId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return respondErr(w, http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post "+attach.PostId))
	}

	return respondJSON(w, commentAdded)
}

func notifyOnFailedAttachment(ji Instance, mattermostUserId, issueKey string, err error, format string, args ...interface{}) {
	msg := "Failed to attach to issue: " + issueKey + ", " + fmt.Sprintf(format, args...)

	ji.GetPlugin().API.LogError(fmt.Sprintf("%s: %v", msg, err), "issue", issueKey)
	errMsg := err.Error()
	if len(errMsg) > 2048 {
		errMsg = errMsg[:2048]
	}
	_, _ = ji.GetPlugin().CreateBotDMtoMMUserId(mattermostUserId,
		"%s. Please notify your system administrator.\n%s", msg, errMsg)
}

func getPermaLink(ji Instance, postId string, currentTeam string) string {
	return fmt.Sprintf("%v/%v/pl/%v", ji.GetPlugin().GetSiteURL(), currentTeam, postId)
}

func (p *Plugin) getIssueDataForCloudWebhook(ji Instance, issueKey string) (*jira.Issue, error) {
	jci, ok := ji.(*jiraCloudInstance)
	if !ok {
		return nil, errors.New("Must be a JIRA Cloud instance, is " + ji.GetType())
	}

	jiraClient, err := jci.getJIRAClientForBot()
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
			return nil, errors.New(`We couldn't find the issue key, or the cloud "bot" client does not have the appropriate permissions to view the issue.`)
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
	case "status":
		return NewStringSet(issue.Fields.Status.ID)
	case "labels":
		return NewStringSet(issue.Fields.Labels...)
	case "priority":
		return NewStringSet(issue.Fields.Priority.ID)
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

func (p *Plugin) getIssueAsSlackAttachment(ji Instance, jiraUser JIRAUser, issueKey string) ([]*model.SlackAttachment, error) {
	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return nil, err
	}

	issue, err := client.GetIssue(issueKey, nil)
	if err != nil {
		switch StatusCode(err) {
		case http.StatusNotFound:
			return nil, errors.New("We couldn't find the issue key, or you do not have the appropriate permissions to view the issue. Please try again or contact your Jira administrator.")

		case http.StatusUnauthorized:
			return nil, errors.New("You do not have the appropriate permissions to view the issue. Please contact your Jira administrator.")

		default:
			return nil, errors.WithMessage(err, "request to Jira failed")

		}
	}

	return parseIssue(client, issue)
}

func (p *Plugin) unassignJiraIssue(mmUserId, issueKey string) (string, error) {
	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return "", err
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mmUserId)
	if err != nil {
		return "", err
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return "", err
	}

	// check for valid issue key
	_, err = client.GetIssue(issueKey, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("We couldn't find the issue key `%s`.  Please confirm the issue key and try again.", issueKey)
		return errorMsg, nil
	}

	if err := client.UpdateAssignee(issueKey, &jira.User{}); err != nil {
		if StatusCode(err) == http.StatusForbidden {
			return "You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.", nil
		}
		return "", err
	}

	permalink := fmt.Sprintf("%v/browse/%v", ji.GetURL(), issueKey)

	msg := fmt.Sprintf("Unassigned Jira issue [%s](%s)", issueKey, permalink)
	return msg, nil
}

const MinUserSearchQueryLength = 3

func (p *Plugin) assignJiraIssue(mmUserId, issueKey, userSearch string) (string, error) {
	ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if err != nil {
		return "", err
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mmUserId)
	if err != nil {
		return "", err
	}

	client, err := ji.GetClient(jiraUser)
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
		errorMsg := fmt.Sprintf("We couldn't find the assignee. Please use a Jira member and try again.")
		return "", fmt.Errorf(errorMsg)
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

	permalink := fmt.Sprintf("%v/browse/%v", ji.GetURL(), issueKey)

	msg := fmt.Sprintf("`%s` assigned to Jira issue [%s](%s)", user.DisplayName, issueKey, permalink)
	return msg, nil
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

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return "", err
	}

	transitions, err := client.GetTransitions(issueKey)
	if err != nil {
		return "", errors.New("We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue.")
	}
	if len(transitions) < 1 {
		return "", errors.New("You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.")
	}

	var transition jira.Transition
	matchingStates := []string{}
	availableStates := []string{}

	potentialState := strings.ToLower(strings.Join(strings.Fields(toState), ""))
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
			toState, strings.Join(availableStates, ", "))

	case 1:
		// proceed

	default:
		return "", errors.Errorf("please be more specific, %q matched several states: %q",
			toState, strings.Join(matchingStates, ", "))
	}

	if err := client.DoTransition(issueKey, transition.ID); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("[%s](%v/browse/%v) transitioned to `%s`",
		issueKey, ji.GetURL(), issueKey, transition.To.Name)
	return msg, nil
}
