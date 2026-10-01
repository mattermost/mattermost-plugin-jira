// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	prefixAdminNotice          = "admin_notice_"
	adminNoticeMissingAPIToken = "missing_api_token_"
	adminListPerPage           = 100
)

const missingAPITokenNotice = ":information_source: **Jira Cloud %s: set an Admin API Token**\n\n" +
	"Without it, channel subscriptions miss comments and new issues from Jira users who haven't connected their account to Mattermost, " +
	"and issue key autolinks can't be set up.\n\n" +
	"To fix it, go to **System Console > Plugins > Jira** and fill in **Admin API Token** and **Admin Email**. " +
	"Create the token at https://id.atlassian.com/manage-profile/security/api-tokens. " +
	"Atlassian API tokens expire after at most one year, so renew it before then."

func (p *Plugin) notifyAdminsIfAPITokenMissing(instance Instance) {
	if p.hasAdminAPIToken() {
		return
	}
	p.notifyAdminsOnce(adminNoticeMissingAPIToken, instance.GetID(),
		fmt.Sprintf(missingAPITokenNotice, instance.GetJiraBaseURL()))
}

func (p *Plugin) hasAdminAPIToken() bool {
	conf := p.getConfig()
	return conf.AdminAPIToken != "" && conf.AdminEmail != ""
}

func (p *Plugin) notifyAdminsOnce(notice string, instanceID types.ID, message string) {
	if !p.markOnce(prefixAdminNotice+notice, instanceID) {
		return
	}

	adminIDs, err := p.listSystemAdminIDs()
	if err != nil {
		p.client.Log.Warn("Failed to list system admins for admin notice", "notice", notice, "error", err.Error())
		return
	}
	for _, adminID := range adminIDs {
		if _, err := p.CreateBotDMtoMMUserID(adminID, "%s", message); err != nil {
			p.client.Log.Warn("Failed to send admin notice", "notice", notice, "userID", adminID, "error", err.Error())
		}
	}
}

// markOnce reports whether this is the first time prefix was marked for the
// instance, across all plugin nodes.
func (p *Plugin) markOnce(prefix string, instanceID types.ID) bool {
	firstTime, err := p.client.KV.Set(hashkey(prefix, instanceID.String()), []byte("1"), pluginapi.SetAtomic(nil))
	if err != nil {
		p.client.Log.Warn("Failed to record one-time marker", "prefix", prefix, "instance", instanceID.String(), "error", err.Error())
		return false
	}
	return firstTime
}

func (p *Plugin) listSystemAdminIDs() ([]string, error) {
	var ids []string
	for page := 0; ; page++ {
		users, err := p.client.User.List(&model.UserGetOptions{
			Role:    model.SystemAdminRoleId,
			Active:  true,
			Page:    page,
			PerPage: adminListPerPage,
		})
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			if !user.IsBot {
				ids = append(ids, user.Id)
			}
		}
		if len(users) < adminListPerPage {
			return ids, nil
		}
	}
}
