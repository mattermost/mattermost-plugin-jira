// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

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
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	expiredTokenNotificationCooldown  = 5 * time.Minute
	expiredTokenNotificationKeyPrefix = "expired_token_dm:"
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

func (p *Plugin) disconnectUserDueToExpiredToken(mattermostUserID types.ID, instanceID types.ID) {
	_, disconnectErr := p.DisconnectUser(instanceID.String(), mattermostUserID)
	if disconnectErr != nil && errors.Cause(disconnectErr) == kvstore.ErrNotFound {
		disconnectErr = nil
	}
	if disconnectErr != nil {
		p.client.Log.Warn("Failed to disconnect user after token expiry",
			"mattermostUserID", mattermostUserID,
			"instanceID", instanceID,
			"error", disconnectErr.Error())
	}

	dmKey := expiredTokenNotificationKeyPrefix + mattermostUserID.String() + ":" + instanceID.String()
	ok, kvErr := p.client.KV.Set(dmKey, []byte("1"),
		pluginapi.SetAtomic(nil),
		pluginapi.SetExpiry(expiredTokenNotificationCooldown),
	)
	if kvErr != nil {
		p.client.Log.Warn("Failed to set expired-token notification dedup marker; proceeding without dedup",
			"mattermostUserID", mattermostUserID,
			"instanceID", instanceID,
			"error", kvErr.Error())
	} else if !ok {
		return
	}

	var notifyErr error
	if disconnectErr != nil {
		_, notifyErr = p.CreateBotDMtoMMUserID(mattermostUserID.String(),
			":warning: Your Jira connection has expired. Please manually disconnect and reconnect your account using:\n"+
				"1. `/jira disconnect %s`\n"+
				"2. `/jira connect %s`",
			instanceID, instanceID)
	} else {
		_, notifyErr = p.CreateBotDMtoMMUserID(mattermostUserID.String(),
			":warning: Your Jira connection has expired. Please reconnect your account using `/jira connect %s`.",
			instanceID)
	}
	if notifyErr != nil {
		label := "Failed to send token expiry notification to user"
		if disconnectErr != nil {
			label = "Failed to send token expiry notification to user after disconnect failure"
		}
		p.client.Log.Warn(label,
			"mattermostUserID", mattermostUserID,
			"error", notifyErr.Error())
	}
}

func (p *Plugin) replaceJiraAccountIds(instanceID types.ID, body string, jiraClient Client) string {
	result := body
	isCloud := false
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err == nil {
		isCloud = instance.Common().IsCloudInstance()
	}

	for _, uname := range parseJIRAUsernamesFromText(body) {
		jiraUserIDOrName := uname
		if strings.HasPrefix(uname, "accountid:") {
			jiraUserIDOrName = uname[len("accountid:"):]
		}

		jiraMention := "[~" + uname + "]"

		mattermostUserID, err := p.userStore.LoadMattermostUserID(instanceID, jiraUserIDOrName)
		if err == nil {
			user, userErr := p.client.User.Get(string(mattermostUserID))
			if userErr == nil {
				result = strings.ReplaceAll(result, jiraMention, "@"+user.Username)
				continue
			}
		}

		displayName := p.getJiraUserDisplayName(jiraClient, isCloud, jiraUserIDOrName)
		if displayName != "" {
			result = strings.ReplaceAll(result, jiraMention, displayName)
		} else {
			result = strings.ReplaceAll(result, jiraMention, jiraUserIDOrName)
		}
	}

	return result
}

func (p *Plugin) getJiraUserDisplayName(jiraClient Client, isCloud bool, userIdentifier string) string {
	if jiraClient == nil {
		return ""
	}

	var params map[string]string
	if isCloud {
		params = map[string]string{"accountId": userIdentifier}
	} else {
		params = map[string]string{"username": userIdentifier}
	}

	var user jira.User
	if err := jiraClient.RESTGet("2/user", params, &user); err != nil {
		p.client.Log.Debug("Failed to fetch Jira user for display name", "userIdentifier", userIdentifier, "error", err.Error())
		return ""
	}

	return user.DisplayName
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
	cfg := p.getConfig()
	if cfg.AdminEmail == "" {
		return errors.New("admin email/username is empty in plugin config")
	}
	if cfg.AdminAPIToken == "" {
		return errors.New("admin API token is empty in plugin config")
	}

	jsonBytes, err := decrypt([]byte(cfg.AdminAPIToken), []byte(cfg.EncryptionKey))
	if err != nil {
		p.client.Log.Warn("Error decrypting admin API token; re-save the Admin API Token in System Console", "error", err.Error())
		return err
	}
	var adminAPIToken string
	if err = json.Unmarshal(jsonBytes, &adminAPIToken); err != nil {
		p.client.Log.Warn("Error unmarshalling admin API token", "error", err.Error())
		return err
	}
	if adminAPIToken == "" {
		return errors.New("decrypted admin API token is empty; re-save the Admin API Token in System Console")
	}

	encodedAuth := base64.StdEncoding.EncodeToString([]byte(cfg.AdminEmail + ":" + adminAPIToken))
	req.Header.Set("Authorization", "Basic "+encodedAuth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mattermost-Plugin-Jira/"+manifest.Version)

	return nil
}
