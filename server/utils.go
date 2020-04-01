// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"regexp"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

func (p *Plugin) CreateBotDMPost(ji Instance, userId, message, postType string) (post *model.Post, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr,
				fmt.Sprintf("failed to create direct post to user %v: ", userId))
		}
	}()

	// Don't send DMs to users who have turned off notifications
	jiraUser, err := p.userStore.LoadJIRAUser(ji, userId)
	if err != nil {
		// not connected to Jira, so no need to send a DM, and no need to report an error
		return nil, nil
	}
	if jiraUser.Settings == nil || !jiraUser.Settings.Notifications {
		return nil, nil
	}

	conf := p.getConfig()
	channel, appErr := p.API.GetDirectChannel(userId, conf.botUserID)
	if appErr != nil {
		return nil, appErr
	}

	post = &model.Post{
		UserId:    conf.botUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	_, appErr = p.API.CreatePost(post)
	if appErr != nil {
		return nil, appErr
	}

	return post, nil
}

func (p *Plugin) CreateBotDMtoMMUserId(mattermostUserId, format string, args ...interface{}) (post *model.Post, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr,
				fmt.Sprintf("failed to create DMError to user %v: ", mattermostUserId))
		}
	}()

	conf := p.getConfig()
	channel, appErr := p.API.GetDirectChannel(mattermostUserId, conf.botUserID)
	if appErr != nil {
		return nil, appErr
	}

	post = &model.Post{
		UserId:    conf.botUserID,
		ChannelId: channel.Id,
		Message:   fmt.Sprintf(format, args...),
	}

	_, appErr = p.API.CreatePost(post)
	if appErr != nil {
		return nil, appErr
	}

	return post, nil
}

func (p *Plugin) StoreCurrentJIRAInstanceAndNotify(ji Instance) error {
	appErr := p.currentInstanceStore.StoreCurrentJIRAInstance(ji)
	if appErr != nil {
		return appErr
	}
	// Notify users we have installed an instance
	p.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": true,
			"instance_type":      ji.GetType(),
		},
		&model.WebsocketBroadcast{},
	)
	return nil
}

func replaceJiraAccountIds(ji Instance, body string) string {
	result := body

	for _, uname := range parseJIRAUsernamesFromText(body) {
		if !strings.HasPrefix(uname, "accountid:") {
			continue
		}

		jiraUserID := uname[len("accountid:"):]
		jiraUser, err := ji.GetPlugin().userStore.LoadJIRAUserByAccountId(ji, jiraUserID)
		if err != nil {
			continue
		}

		if jiraUser.DisplayName != "" {
			result = strings.ReplaceAll(result, uname, jiraUser.DisplayName)
		}
	}

	return result
}

func (p *Plugin) loadJIRAProjectKeys(jiraClient *jira.Client) ([]string, error) {
	list, _, err := jiraClient.Project.GetList()
	if err != nil {
		return nil, errors.WithMessage(err, "Error requesting list of Jira projects")
	}

	projectKeys := []string{}
	for _, proj := range *list {
		projectKeys = append(projectKeys, proj.Key)
	}
	return projectKeys, nil
}

func parseJIRAUsernamesFromText(text string) []string {
	usernameMap := map[string]bool{}
	usernames := []string{}

	var re = regexp.MustCompile(`(?m)\[~([a-zA-Z0-9-_.:\+]+)\]`)
	for _, match := range re.FindAllString(text, -1) {
		name := match[:len(match)-1]
		name = name[2:]
		if !usernameMap[name] {
			usernames = append(usernames, name)
			usernameMap[name] = true
		}
	}

	return usernames
}

func parseJIRAIssuesFromText(text string, keys []string) []string {
	issueMap := map[string]bool{}
	issues := []string{}

	for _, key := range keys {
		var re = regexp.MustCompile(fmt.Sprintf(`(?m)%s-[0-9]+`, key))
		for _, match := range re.FindAllString(text, -1) {
			if !issueMap[match] {
				issues = append(issues, match)
				issueMap[match] = true
			}
		}
	}

	return issues
}

func respondEphemeralErr(plugin *Plugin, status int, e error, mmuserID, jiraBotID, channelID, message string) (int, error) {
	_ = plugin.API.SendEphemeralPost(mmuserID, makePost(jiraBotID, channelID, message))
	if e != nil {
		return status, e
	}
	return status, errors.New(message)
}

func makePost(userId, channelId, message string) *model.Post {
	return &model.Post{
		UserId:    userId,
		ChannelId: channelId,
		Message:   message,
	}
}
