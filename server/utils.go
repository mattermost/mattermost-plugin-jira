// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"

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
	channel, err := p.client.Channel.GetDirect(mattermostUserID.String(), conf.botUserID)
	if err != nil {
		return nil, err
	}

	post = &model.Post{
		UserId:    conf.botUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	err = p.client.Post.CreatePost(post)
	if err != nil {
		return nil, err
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
	channel, err := p.client.Channel.GetDirect(mattermostUserID, conf.botUserID)
	if err != nil {
		return nil, err
	}

	post = &model.Post{
		UserId:    conf.botUserID,
		ChannelId: channel.Id,
		Message:   fmt.Sprintf(format, args...),
	}

	err = p.client.Post.CreatePost(post)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (p *Plugin) replaceJiraAccountIds(instanceID types.ID, body string) string {
	result := body
	for _, uname := range parseJIRAUsernamesFromText(body) {
		jiraUserIDOrName := ""
		if strings.HasPrefix(uname, "accountid:") {
			jiraUserIDOrName = uname[len("accountid:"):]
		} else {
			jiraUserIDOrName = uname
		}

		mattermostUserID, err := p.userStore.LoadMattermostUserID(instanceID, jiraUserIDOrName)
		if err != nil {
			continue
		}

		user, err := p.client.User.Get(string(mattermostUserID))
		if err != nil {
			continue
		}

		jiraUserName := "[~" + uname + "]"
		result = strings.ReplaceAll(result, jiraUserName, "@"+user.Username)
	}

	return result
}

func parseJIRAUsernamesFromText(text string) []string {
	usernameMap := map[string]bool{}
	usernames := []string{}

	var re = regexp.MustCompile(`(?m)\[~([a-zA-Z0-9-_@.:\+]+)\]`)
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

// getS256PKCEParams creates the code_challenge and code_verifier params for oauth2
func getS256PKCEParams() (*PKCEParams, error) {
	buf := make([]byte, PKCEByteArrayLength)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}

	verifier := base64.RawURLEncoding.EncodeToString(buf)

	h := sha256.New()
	h.Write([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return &PKCEParams{
		CodeChallenge: challenge,
		CodeVerifier:  verifier,
	}, nil
}

func (p *Plugin) SetAdminAPITokenRequestHeader(req *http.Request) error {
	encryptedAdminAPIToken := p.getConfig().AdminAPIToken
	jsonBytes, err := decrypt([]byte(encryptedAdminAPIToken), []byte(p.getConfig().EncryptionKey))
	if err != nil {
		p.client.Log.Warn("Error decrypting admin API token", "error", err.Error())
		return err
	}
	var adminAPIToken string
	err = json.Unmarshal(jsonBytes, &adminAPIToken)
	if err != nil {
		p.client.Log.Warn("Error unmarshalling admin API token", "error", err.Error())
		return err
	}

	encodedAuth := base64.StdEncoding.EncodeToString([]byte(p.getConfig().AdminEmail + ":" + adminAPIToken))
	req.Header.Set("Authorization", "Basic "+encodedAuth)
	req.Header.Set("Accept", "application/json")

	return nil
}
