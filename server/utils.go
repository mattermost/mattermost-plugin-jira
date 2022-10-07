// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func (p *Plugin) CreateBotDMPost(instanceID, mattermostUserID types.ID, message, postType string) (post *model.Post, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr,
				fmt.Sprintf("failed to create direct post to user %v: ", mattermostUserID))
		}
	}()

	// Don't send DMs to users who have turned off notifications
	c, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		// not connected to Jira, so no need to send a DM, and no need to report an error
		return nil, nil
	}
	if c.Settings == nil || !c.Settings.Notifications {
		return nil, nil
	}

	conf := p.getConfig()
	channel, appErr := p.API.GetDirectChannel(mattermostUserID.String(), conf.botUserID)
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

func (p *Plugin) CreateBotDMtoMMUserID(mattermostUserID, format string, args ...interface{}) (post *model.Post, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr,
				fmt.Sprintf("failed to create DMError to user %v: ", mattermostUserID))
		}
	}()

	conf := p.getConfig()
	channel, appErr := p.API.GetDirectChannel(mattermostUserID, conf.botUserID)
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

func (p *Plugin) replaceJiraAccountIds(instanceID types.ID, body string) string {
	result := body

	for _, uname := range parseJIRAUsernamesFromText(body) {
		if !strings.HasPrefix(uname, "accountid:") {
			continue
		}

		jiraUserID := uname[len("accountid:"):]
		mattermostUserID, err := p.userStore.LoadMattermostUserID(instanceID, jiraUserID)
		if err != nil {
			continue
		}
		c, err := p.userStore.LoadConnection(instanceID, mattermostUserID)
		if err != nil {
			continue
		}

		if c.DisplayName != "" {
			result = strings.ReplaceAll(result, uname, c.DisplayName)
		}
	}

	return result
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

func isImageMIME(mime string) bool {
	return strings.HasPrefix(mime, "image")
}

func isEmbbedableMIME(mime string) bool {
	validMimes := [...]string{
		// .swf
		"application/x-shockwave-flash",
		// .mov
		"video/quicktime",
		// .rm
		"application/vnd.rn-realmedia",
		// .ram
		"audio/x-pn-realaudio",
		// .mp3
		"audio/mpeg3",
		"audio/x-mpeg-3",
		"video/mpeg",
		"video/x-mpeg",
		// .mp4
		"video/mp4",
		// .wmv
		"video/x-ms-wmv",
		"video/x-ms-asf",
		// .wma
		"audio/x-ms-wma",
	}
	for _, validMime := range validMimes {
		if mime == validMime {
			return true
		}
	}
	return false
}
