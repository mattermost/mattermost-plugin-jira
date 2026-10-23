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

// hasAdminAPIToken reports whether the plugin is configured with enough
// Admin API Token settings to authenticate against instance. Jira Cloud
// requires both the token and the admin email (used for Basic auth); Jira
// Server/Data Center only requires the token, since it's sent as a Bearer PAT.
func (p *Plugin) hasAdminAPIToken(instance Instance) bool {
	conf := p.getConfig()
	if conf.AdminAPIToken == "" {
		return false
	}
	if instance.Common().IsCloudInstance() {
		return conf.AdminEmail != ""
	}
	return true
}

// doAdminAPITokenRequest sends req authenticated with the Admin API Token
// configured in plugin settings. Jira Cloud expects HTTP Basic auth with the
// account email and API token:
// https://developer.atlassian.com/cloud/jira/platform/basic-auth-for-rest-apis/
// Jira Server/Data Center personal access tokens must be sent as a Bearer
// token instead; Basic auth would be interpreted as a username/password login
// attempt, which fails (and can trigger account lockouts) with a PAT:
// https://confluence.atlassian.com/enterprise/using-personal-access-tokens-1026032365.html
// Server/DC installs configured with a username and password (e.g. Jira
// Server before 8.14, which has no PATs) get a 401 for the Bearer request,
// so it's retried with Basic auth when an admin username is set. If that retry
// is also rejected, retries stop until the plugin configuration changes.
func (p *Plugin) doAdminAPITokenRequest(client *http.Client, req *http.Request, instance Instance) (*http.Response, error) {
	isCloud := instance.Common().IsCloudInstance()
	if err := p.setAdminAPITokenRequestHeader(req, isCloud); err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil || isCloud || resp.StatusCode != http.StatusUnauthorized ||
		p.getConfig().AdminEmail == "" || p.adminBasicFallbackFailed.Load() {
		return resp, err
	}
	resp.Body.Close()

	if err = p.setAdminAPITokenRequestHeader(req, true); err != nil {
		return nil, err
	}
	resp, err = client.Do(req)
	if err == nil && resp.StatusCode == http.StatusUnauthorized && p.adminBasicFallbackFailed.CompareAndSwap(false, true) {
		p.client.Log.Warn("Admin API Token was rejected as both a Bearer token and a Basic auth password; "+
			"Basic auth retries are disabled until the plugin configuration is saved again",
			"seraphLoginReason", resp.Header.Get("X-Seraph-LoginReason"))
	}
	return resp, err
}

func (p *Plugin) setAdminAPITokenRequestHeader(req *http.Request, useBasicAuth bool) error {
	cfg := p.getConfig()
	if useBasicAuth && cfg.AdminEmail == "" {
		return errors.New("admin email is empty in plugin config (required for Jira Cloud)")
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

	if useBasicAuth {
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(cfg.AdminEmail + ":" + adminAPIToken))
		req.Header.Set("Authorization", "Basic "+encodedAuth)
	} else {
		req.Header.Set("Authorization", "Bearer "+adminAPIToken)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mattermost-Plugin-Jira/"+manifest.Version)

	return nil
}
