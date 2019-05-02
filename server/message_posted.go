// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	var err error
	defer func() {
		if err != nil {
			p.errorf("MessageHasBeenPosted: %v", err)
		}
	}()

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		return
	}

	jiraUser, err := p.LoadJIRAUser(ji, post.UserId)
	if err != nil {
		return
	}

	jiraClient, err := ji.GetJIRAClient(jiraUser)
	if err != nil {
		return
	}

	projectKeys, err := p.loadJIRAProjectKeys(jiraClient)
	if err != nil {
		return
	}

	issues := parseJIRAIssuesFromText(post.Message, projectKeys)
	if len(issues) == 0 {
		return
	}

	channel, appErr := p.API.GetChannel(post.ChannelId)
	if appErr != nil {
		err = errors.WithMessagef(appErr, "failed to load channel ID: %s", post.ChannelId)
		return
	}

	if channel.Type != model.CHANNEL_OPEN {
		err = errors.New("ignoring Jira comment in " + channel.Name)
		return
	}

	team, appErr := p.API.GetTeam(channel.TeamId)
	if appErr != nil {
		err = errors.WithMessagef(appErr, "failed to load team ID: %v", channel.TeamId)
		return
	}

	user, appErr := p.API.GetUser(post.UserId)
	if appErr != nil {
		err = errors.WithMessagef(appErr, "failed to load user ID: %v", post.UserId)
		return
	}

	conf := p.API.GetConfig()
	permalink := fmt.Sprintf("%v/%v/pl/%v", *conf.ServiceSettings.SiteURL, team.Name, post.Id)

	for _, issue := range issues {
		comment := &jira.Comment{
			Body: fmt.Sprintf("%s mentioned this ticket in Mattermost:\n{quote}\n%s\n{quote}\n\n[View message in Mattermost|%s]",
				user.Username, post.Message, permalink),
		}

		_, _, err := jiraClient.Issue.AddComment(issue, comment)
		if err != nil {
			p.errorf("MessageHasBeenPosted: failed to add the comment to Jira, error: %v", err)
		}
	}
}
