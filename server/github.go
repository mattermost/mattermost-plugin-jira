package main

import (
	"errors"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/google/go-github/v25/github"
	"io/ioutil"
	"net/http"
	"strings"
)

func httpGithubEvent(ji Instance, w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		return http.StatusUnauthorized, errors.New("Not authorized")
	}

	jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	client, err := ji.GetClient(jiraUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest, err
	}

	event, _ := github.ParseWebHook(github.WebHookType(r), body)
	plugin := ji.GetPlugin()

	switch event := event.(type) {
	case *github.PullRequestEvent:
		plugin.askAssigneeToClosePRRelatedIssues(client, event)
	}

	return http.StatusOK, nil
}

func (p *Plugin) askAssigneeToClosePRRelatedIssues(client Client, event *github.PullRequestEvent) error {
	instance, error := p.currentInstanceStore.LoadCurrentJIRAInstance()
	if error != nil {
		return error
	}

	issues := findIssuesInLinks(*event.PullRequest.Body, instance.GetURL())

	jiraIssues, error := client.SearchIssues(fmt.Sprintf("id IN(%s)", strings.Join(issues, ",")), nil)
	if error != nil {
		return error
	}

	jiraIssuesByAssignee := map[string][]jira.Issue{}
	for _, jiraIssue := range jiraIssues {
		assigneeMattermostId, error := p.userStore.LoadMattermostUserId(instance, jiraIssue.Fields.Assignee.AccountID)
		if error != nil {
			continue
		}

		jiraIssuesForAssignee := jiraIssuesByAssignee[assigneeMattermostId]
		if jiraIssuesForAssignee == nil {
			jiraIssuesByAssignee[assigneeMattermostId] = []jira.Issue{jiraIssue}
		} else {
			jiraIssuesByAssignee[assigneeMattermostId] = append(jiraIssuesForAssignee, jiraIssue)
		}
	}

	for assigneeMattermostId, jiraIssues := range jiraIssuesByAssignee {
		jiraIssuesLink := ""
		for _, jiraIssue := range jiraIssues {
			jiraIssuesLink += ", " + mdKeySummaryLink(&jiraIssue)
		}

		markdownFormatLink := formatMarkdownLink(*event.PullRequest.Title, *event.PullRequest.HTMLURL)
		p.CreateBotDMPost(instance, assigneeMattermostId,
			fmt.Sprintf("The PR %s has been merged. Do you want to close the linked Jira issue(s) %s", markdownFormatLink, jiraIssuesLink), "")
	}

	return nil
}
