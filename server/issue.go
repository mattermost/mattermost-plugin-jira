// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
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
	if post != nil && post.ParentId != "" {
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

	project, _, err := jiraClient.Project.Get(issue.Fields.Project.Key)
	if err != nil {
		err = userFriendlyJiraError(nil, err)
		return http.StatusInternalServerError, errors.WithMessagef(err,
			"failed to get project %q", issue.Fields.Project.Key)
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
			ChannelId: channelId,
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
		err = userFriendlyJiraError(resp, err)
		// if have an error and Jira tells us there are required fields send user
		// link to jira with fields already filled in.  Note the user will also see
		// these errors in Jira.
		// Note that RequiredFieldsNotCovered is also empty
		if strings.Contains(err.Error(), "is required.") {
			message := fmt.Sprintf("Failed to create issue. Your Jira project requires fields the plugin does not yet support. "+
				"[Please create your Jira issue manually](%s) or contact your Jira administrator.\n%v",
				buildCreateQuery(ji, project, issue).URL.String(),
				err)

			_ = api.SendEphemeralPost(mattermostUserId, &model.Post{
				Message:   message,
				ChannelId: channelId,
				RootId:    rootId,
				ParentId:  parentId,
				UserId:    ji.GetPlugin().getConfig().botUserID,
			})
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "{}")
			return http.StatusOK, nil
		}

		return http.StatusInternalServerError, errors.Errorf("Failed to create issue. %s", err.Error())
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", ji.GetURL(), created.Key),
		ChannelId: channelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(appErr, "failed to create notification post "+create.PostId)
	}

	if post != nil && len(post.FileIds) > 0 {
		go func() {
			conf := ji.GetPlugin().getConfig()
			for _, fileId := range post.FileIds {
				mattermostName, _, e := attachFileToIssue(api, jiraClient, created.ID, fileId, conf.maxAttachmentSize)
				if e != nil {
					notifyOnFailedAttachment(ji, mattermostUserId, created.Key, e, "file: %s", mattermostName)
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

func httpAPIGetCreateIssueMetadataForProjects(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("Request: " + r.Method + " is not allowed, must be GET")
	}

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("not authorized")
	}

	projectKeys := r.FormValue("project-keys")
	if projectKeys == "" {
		return http.StatusBadRequest, errors.New("project-keys query param is required")
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
		ProjectKeys: projectKeys,
	})
	if err != nil {
		err = userFriendlyJiraError(resp, err)
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to GetCreateIssueMetadata")
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
		err = userFriendlyJiraError(resp, err)
		return http.StatusInternalServerError,
			errors.WithMessage(err, "failed to GetCreateIssueMetadata")
	}

	w.Header().Set("Content-Type", "application/json")

	// Generic option, used in the options list in react-select
	type option struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	type projectMetadata struct {
		Projects          []option            `json:"projects"`
		IssuesPerProjects map[string][]option `json:"issues_per_project"`
	}

	var bb []byte
	if len(cimd.Projects) == 0 {
		bb = []byte(`{"error": "You do not have permission to create issues in any projects. Please contact your Jira admin."}`)
	} else {
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

	searchRes, _, err := jiraClient.Issue.Search(jqlString, &jira.SearchOptions{
		MaxResults: 50,
		Fields:     []string{"key", "summary"},
	})
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(userFriendlyJiraError(nil, err),
				"failed to get search results")
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

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message|%s] *from @%s*\n", jiraUser.User.DisplayName, permalink, commentUser.Username)

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

	go func() {
		conf := ji.GetPlugin().getConfig()
		extraText := ""
		for _, fileId := range post.FileIds {
			mattermostName, jiraName, e := attachFileToIssue(api, jiraClient, attach.IssueKey, fileId, conf.maxAttachmentSize)
			if e != nil {
				notifyOnFailedAttachment(ji, mattermostUserId, attach.IssueKey, e, "file: %s", mattermostName)
			}

			extraText += "\n\nAttachment: !" + jiraName + "!"
		}
		if extraText == "" {
			return
		}

		jiraComment.ID = commentAdded.ID
		jiraComment.Body += extraText
		_, _, err = jiraClient.Issue.UpdateComment(attach.IssueKey, &jiraComment)
		if err != nil {
			notifyOnFailedAttachment(ji, mattermostUserId, attach.IssueKey, err, "failed to completely update comment with attachments")
		}
	}()

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

func notifyOnFailedAttachment(ji Instance, mattermostUserId, issueKey string, err error, format string, args ...interface{}) {
	msg := "Failed to attach to issue: " + issueKey + ", " + fmt.Sprintf(format, args...)

	ji.GetPlugin().API.LogError(fmt.Sprintf("%s: %v", msg, err), "issue", issueKey)
	_, _ = ji.GetPlugin().CreateBotDMtoMMUserId(mattermostUserId,
		"%s. Please notify your system administrator.\n- Error: %v", msg, err)
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
		switch {
		case resp == nil:
			return nil, errors.WithMessage(userFriendlyJiraError(nil, err),
				"request to Jira failed")

		case resp.StatusCode == http.StatusNotFound:
			return nil, errors.New("We couldn't find the issue key, or you do not have the appropriate permissions to view the issue. Please try again or contact your Jira administrator.")

		case resp.StatusCode == http.StatusUnauthorized:
			return nil, errors.New("You do not have the appropriate permissions to view the issue. Please contact your Jira administrator.")
		}
	}

	return parseIssue(issue), nil
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

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return "", err
	}

	// required minimum of three letters in assignee value
	if len(userSearch) < MinUserSearchQueryLength {
		errorMsg := fmt.Sprintf("`%s` contains less than %v characters.", userSearch, MinUserSearchQueryLength)
		return errorMsg, nil
	}

	// check for valid issue key
	_, _, err = jiraClient.Issue.Get(issueKey, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("We couldn't find the issue key `%s`.  Please confirm the issue key and try again.", issueKey)
		return errorMsg, nil
	}

	// Get list of assignable users
	var jiraUsers []jira.User
	status, err := JiraGet(jiraClient, "user/assignable/search",
		map[string]string{
			"issueKey":   issueKey,
			"username":   userSearch,
			"maxResults": "10",
		}, &jiraUsers)
	if status == 401 {
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

	if _, err := jiraClient.Issue.UpdateAssignee(issueKey, &user); err != nil {
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

	var transition jira.Transition
	matchingStates := []string{}
	availableStates := []string{}
	for _, t := range transitions {
		if strings.Contains(strings.ToLower(t.To.Name), strings.ToLower(toState)) {
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

	if _, err := jiraClient.Issue.DoTransition(issueKey, transition.ID); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("[%s](%v/browse/%v) transitioned to `%s`",
		issueKey, ji.GetURL(), issueKey, transition.To.Name)
	return msg, nil
}

func userFriendlyJiraError(resp *jira.Response, err error) error {
	jerr, ok := err.(*jira.Error)
	if !ok {
		if resp == nil {
			return err
		}
		// Closes resp.Body()
		err = jira.NewJiraError(resp, err)
		jerr, ok = err.(*jira.Error)
		if !ok {
			return err
		}
	}
	if len(jerr.Errors) == 0 && len(jerr.ErrorMessages) == 0 {
		return err
	}

	message := ""
	for k, v := range jerr.Errors {
		message += fmt.Sprintf(" - %s: %s\n", k, v)
	}
	for _, m := range jerr.ErrorMessages {
		message += fmt.Sprintf(" - %s\n", m)
	}
	return errors.New(message)
}

// Upload file attachments in the background
func attachFileToIssue(api plugin.API, jiraClient *jira.Client, issueKey, fileId string, maxSize ByteSize) (mattermostName, jiraName string, err error) {
	fileinfo, appErr := api.GetFileInfo(fileId)
	if appErr != nil {
		return "", "", appErr
	}
	if ByteSize(fileinfo.Size) > maxSize {
		return fileinfo.Name, "", errors.Errorf("Maximum attachment size %v exceeded, file size %v", maxSize, ByteSize(fileinfo.Size))
	}
	fileBytes, appErr := api.ReadFile(fileinfo.Path)
	if appErr != nil {
		return fileinfo.Name, "", appErr
	}

	attachments, _, err := jiraClient.Issue.PostAttachment(issueKey, bytes.NewReader(fileBytes), fileinfo.Name)
	if err != nil {
		return fileinfo.Name, "", err
	}
	if attachments == nil || len(*attachments) == 0 {
		return fileinfo.Name, "", errors.New("unreachable error, attaching file" + fileinfo.Name)
	}
	// There will only ever be one attachment at a time.
	attachment := (*attachments)[0]
	return fileinfo.Name, attachment.Filename, nil
}

func JiraGet(jiraClient *jira.Client, api string, params map[string]string, dest interface{}) (int, error) {
	return jiraGet(jiraClient, 2, api, params, dest)
}

func JiraGet2(jiraClient *jira.Client, api string, params map[string]string, dest interface{}) (int, error) {
	return jiraGet(jiraClient, 2, api, params, dest)
}

func jiraGet(jiraClient *jira.Client, version int, api string, params map[string]string, dest interface{}) (int, error) {
	apiEndpoint := fmt.Sprintf("/rest/api/%v/%s", version, api)
	req, err := jiraClient.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := jiraClient.Do(req, dest)
	if err != nil {
		err = userFriendlyJiraError(resp, err)
	}
	return resp.StatusCode, err
}
