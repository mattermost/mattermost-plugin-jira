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
	"net/url"
	"regexp"
	"strings"
	"time"

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

func (p *Plugin) notifyUserTokenExpired(mattermostUserID types.ID, instanceID types.ID) {
	_, err := p.CreateBotDMtoMMUserID(mattermostUserID.String(),
		":warning: Your Jira connection has expired. Please reconnect your account using `/jira connect %s`.",
		instanceID)
	if err != nil {
		p.client.Log.Warn("Failed to send token expiry notification to user",
			"mattermostUserID", mattermostUserID,
			"error", err.Error())
	}
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

		jiraMention := "[~" + uname + "]"

		mattermostUserID, err := p.userStore.LoadMattermostUserID(instanceID, jiraUserIDOrName)
		if err == nil {
			user, userErr := p.client.User.Get(string(mattermostUserID))
			if userErr == nil {
				result = strings.ReplaceAll(result, jiraMention, "@"+user.Username)
				continue
			}
		}

		displayName := p.getJiraUserDisplayName(instanceID, jiraUserIDOrName)
		if displayName != "" {
			result = strings.ReplaceAll(result, jiraMention, displayName)
		} else {
			result = strings.ReplaceAll(result, jiraMention, jiraUserIDOrName)
		}
	}

	return result
}

func (p *Plugin) getJiraUserDisplayName(instanceID types.ID, userIdentifier string) string {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		p.client.Log.Debug("Failed to load instance for user display name lookup", "instanceID", instanceID.String(), "error", err.Error())
		return ""
	}

	baseURL := instance.GetJiraBaseURL()
	if baseURL == "" {
		p.client.Log.Debug("Instance has empty URL", "instanceID", instanceID.String())
		return ""
	}

	var userURL string
	if instance.Common().IsCloudInstance() {
		userURL = fmt.Sprintf("%s/rest/api/2/user?accountId=%s", baseURL, url.QueryEscape(userIdentifier))
	} else {
		userURL = fmt.Sprintf("%s/rest/api/2/user?username=%s", baseURL, url.QueryEscape(userIdentifier))
	}

	req, err := http.NewRequest(http.MethodGet, userURL, nil)
	if err != nil {
		p.client.Log.Debug("Failed to create request for Jira user lookup", "error", err.Error())
		return ""
	}

	if err := p.SetAdminAPITokenRequestHeader(req); err != nil {
		return ""
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		p.client.Log.Debug("Failed to fetch Jira user", "userIdentifier", userIdentifier, "error", err.Error())
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.client.Log.Debug("Jira user lookup returned non-OK status", "userIdentifier", userIdentifier, "status", resp.StatusCode)
		return ""
	}

	var jiraUser struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jiraUser); err != nil {
		p.client.Log.Debug("Failed to decode Jira user response", "userIdentifier", userIdentifier, "error", err.Error())
		return ""
	}

	return jiraUser.DisplayName
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
