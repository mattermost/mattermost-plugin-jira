// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"math"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

type ByteSize int64

const sizeB = ByteSize(1)
const sizeKb = 1024 * sizeB
const sizeMb = 1024 * sizeKb
const sizeGb = 1024 * sizeMb
const sizeTb = 1024 * sizeGb

var sizeUnits = []ByteSize{sizeTb, sizeGb, sizeMb, sizeKb, sizeB}
var sizeSuffixes = []string{"Tb", "Gb", "Mb", "Kb", "b"}

func (size ByteSize) String() string {
	if size == 0 {
		return "0"
	}

	withCommas := func(in string) string {
		out := ""
		for len(in) > 3 {
			out = "," + in[len(in)-3:] + out
			in = in[:len(in)-3]
		}
		out = in + out
		return out
	}

	for i, u := range sizeUnits {
		if size < u {
			continue
		}
		if u == sizeB {
			return withCommas(strconv.FormatUint(uint64(size), 10)) + sizeSuffixes[i]
		}

		if size > math.MaxInt64/10 {
			return "n/a"
		}

		s := strconv.FormatUint(uint64((size*10+u/2)/u), 10)
		l := len(s)
		switch {
		case l < 2:
			return "n/a"
		case s[l-1] == '0':
			return withCommas(s[:l-1]) + sizeSuffixes[i]
		default:
			return withCommas(s[:l-1]) + "." + s[l-1:] + sizeSuffixes[i]
		}
	}
	return "n/a"
}

func ParseByteSize(str string) (ByteSize, error) {
	u := sizeB
	str = strings.ToLower(str)
	for i, s := range sizeSuffixes {
		if strings.HasSuffix(str, strings.ToLower(s)) {
			str = str[:len(str)-len(s)]
			u = sizeUnits[i]
			break
		}
	}

	str = strings.ReplaceAll(str, ",", "")
	n, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return ByteSize(n) * u, nil
	}
	numerr := err.(*strconv.NumError)
	if numerr.Err != strconv.ErrSyntax {
		return 0, err
	}
	fl, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	return ByteSize(fl * float64(u)), nil
}

func normalizeInstallURL(mattermostSiteURL, jiraURL string) (string, error) {
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

	jiraURL = strings.TrimSuffix(u.String(), "/")
	if jiraURL == strings.TrimSuffix(mattermostSiteURL, "/") {
		return "", errors.Errorf("%s is the Mattermost site URL. Please use your Jira URL with `/jira install`.", jiraURL)
	}

	return jiraURL, nil
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

// Reference: https://gobyexample.com/collection-functions
func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}
