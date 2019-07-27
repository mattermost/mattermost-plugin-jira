// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func normalizeInstallURL(jiraURL string) (string, error) {
	u, err := url.Parse(jiraURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		ss := strings.Split(u.Path, "/")
		if len(ss) > 0 && ss[0] != "" {
			u.Host = ss[0]
			u.Path = path.Join(ss[1:]...)
		}
		u, err = url.Parse(u.String())
		if err != nil {
			return "", err
		}
	}
	if u.Host == "" {
		return "", errors.Errorf("Invalid URL, no hostname: %q", jiraURL)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

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
		},
		&model.WebsocketBroadcast{},
	)
	return nil
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

	var re = regexp.MustCompile(`(?m)\[~([a-zA-Z0-9-_.\+]+)\]`)
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
