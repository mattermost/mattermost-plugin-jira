// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
)

func testWebhookRequest(filename string) *http.Request {
	if f, err := os.Open(filepath.Join("testdata", filename)); err != nil {
		panic(err)
	} else {
		return httptest.NewRequest("POST",
			"/webhook?team=theteam&channel=thechannel&secret=thesecret&updated_all=1",
			f)
	}
}

type testWebhookWrapper struct {
	Webhook
	postedToChannel     *model.Post
	postedNotifications []*model.Post
}

func (wh testWebhookWrapper) Events() StringSet {
	return wh.Webhook.Events()
}

func (wh *testWebhookWrapper) PostToChannel(p *Plugin, instanceID types.ID, channelId, fromUserId string) (*model.Post, int, error) {
	post, status, err := wh.Webhook.PostToChannel(p, "", channelId, fromUserId)
	if post != nil {
		wh.postedToChannel = post
	}
	return post, status, err
}
func (wh *testWebhookWrapper) PostNotifications(p *Plugin, instanceID types.ID) ([]*model.Post, int, error) {
	posts, status, err := wh.Webhook.PostNotifications(p, instanceID)
	if len(posts) != 0 {
		wh.postedNotifications = append(wh.postedNotifications, posts...)
	}
	return posts, status, err
}

func TestWebhookHTTP(t *testing.T) {
	validConfiguration := TestConfiguration{
		Secret:   "thesecret",
		UserName: "theuser",
	}

	for name, tc := range map[string]struct {
		Request                 *http.Request
		ExpectedHeadline        string
		ExpectedSlackAttachment bool
		ExpectedText            string
		ExpectedFields          []*model.SlackAttachmentField
		ExpectedStatus          int
		ExpectedIgnored         bool // Indicates that no post was made as a result of the webhook request
		CurrentInstance         bool
	}{
		"issue created": {
			Request:                 testWebhookRequest("webhook-issue-created.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long",
			ExpectedFields: []*model.SlackAttachmentField{
				&model.SlackAttachmentField{
					Title: "Priority",
					Value: "High",
					Short: true,
				},
			},
			CurrentInstance: true,
		},
		"issue created no fields": {
			Request:                 testWebhookRequest("webhook-issue-created-no-relevant-fields.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long",
			ExpectedFields:          []*model.SlackAttachmentField{},
			CurrentInstance:         true,
		},
		"issue created no description": {
			Request:                 testWebhookRequest("webhook-issue-created-no-description.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			// ExpectedText:            "Unit test description, not that long",
			ExpectedFields: []*model.SlackAttachmentField{
				&model.SlackAttachmentField{
					Title: "Priority",
					Value: "High",
					Short: true,
				},
			},
			CurrentInstance: true,
		},
		"issue created no description nor fields": {
			Request:          testWebhookRequest("webhook-issue-created-no-description-nor-relevant-fields.json"),
			ExpectedStatus:   http.StatusOK,
			ExpectedHeadline: "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"issue edited": {
			Request:                 testWebhookRequest("webhook-issue-updated-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **edited** the description of story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long, a little longer now",
			CurrentInstance:         true,
		},
		"SERVER (old version) issue edited (no issue_event_type_name)": {
			Request:                 testWebhookRequest("webhook-server-old-issue-updated-no-event-type-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **edited** the description of story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long, a little longer now",
			CurrentInstance:         true,
		},
		"issue renamed": {
			Request:          testWebhookRequest("webhook-issue-updated-renamed.json"),
			ExpectedHeadline: "Test User **updated** summary from \"Unit test summary\" to \"Unit test summary 1\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:     "",
			CurrentInstance:  true,
		},
		"issue assigned nobody": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned-nobody.json"),
			ExpectedHeadline: "Test User **assigned** _nobody_ to story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"issue assigned": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned.json"),
			ExpectedHeadline: "Test User **assigned** Test User to story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"SERVER (old version) issue assigned (no issue_event_type_name)": {
			Request:          testWebhookRequest("webhook-server-old-issue-updated-no-event-type-assigned.json"),
			ExpectedHeadline: "Test User **assigned** Test User to story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"issue assigned on server": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned-on-server.json"),
			ExpectedHeadline: "Test User **assigned** Test User to improvement [PRJA-37: test](http://some-instance-test.centralus.cloudapp.azure.com:8080/browse/PRJA-37)",
			CurrentInstance:  true,
		},
		"issue attachments": {
			Request:          testWebhookRequest("webhook-issue-updated-attachments.json"),
			ExpectedHeadline: "Test User **attached** [test.gif] to, **removed** attachments [test.json] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"issue fix version": {
			Request:          testWebhookRequest("webhook-issue-updated-fix-version.json"),
			ExpectedHeadline: `Test User **updated** Fix Version from "v1" to "v2" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  true,
		},
		"issue issue type": {
			Request:          testWebhookRequest("webhook-issue-updated-issue-type.json"),
			ExpectedHeadline: `Test User **updated** issuetype from "Task" to "Bug" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  true,
		},
		"issue labels": {
			Request:          testWebhookRequest("webhook-issue-updated-labels.json"),
			ExpectedHeadline: "Test User **added** labels [sad] to, **removed** labels [bad] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"issue lowered priority": {
			Request:          testWebhookRequest("webhook-issue-updated-lowered-priority.json"),
			ExpectedHeadline: `Test User **updated** priority from "High" to "Low" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  true,
		},
		"issue multiple values": {
			Request:                 testWebhookRequest("webhook-issue-updated-multiple-values.json"),
			ExpectedHeadline:        `Test User **updated** story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			ExpectedSlackAttachment: true,
			ExpectedFields: []*model.SlackAttachmentField{
				{
					Title: "",
					Value: "**Fix Version:** ~~v1~~ v2",
					Short: false,
				},
				{
					Title: "",
					Value: "**Assignee:** ~~_nobody_~~ Test User",
					Short: false,
				},
				{
					Title: "",
					Value: "**QA Steps:** ~~None~~ Make sure it does the thing.",
					Short: false,
				},
			},
			CurrentInstance: true,
		},
		"issue raised priority": {
			Request:          testWebhookRequest("webhook-issue-updated-raised-priority.json"),
			ExpectedHeadline: `Test User **updated** priority from "Low" to "High" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  true,
		},
		"issue rank": {
			Request:          testWebhookRequest("webhook-issue-updated-rank.json"),
			ExpectedHeadline: "Test User **updated** Rank from \"~~none~~\" to \"ranked higher\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"issue reopened": {
			Request:                 testWebhookRequest("webhook-issue-updated-reopened.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **updated** story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedFields: []*model.SlackAttachmentField{
				{
					Title: "",
					Value: "**Reopened:** ~~Done~~ Open",
					Short: false,
				},
				{
					Title: "",
					Value: "**Status:** ~~Done~~ To Do",
					Short: false,
				},
			},
			CurrentInstance: true,
		},
		"issue resolved": {
			Request:                 testWebhookRequest("webhook-issue-updated-resolved.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **updated** story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedFields: []*model.SlackAttachmentField{
				{
					Title: "",
					Value: "**Resolved:** ~~Open~~ Done",
					Short: false,
				},
				{
					Title: "",
					Value: "**Status:** ~~In Progress~~ Done",
					Short: false,
				},
			},
			CurrentInstance: true,
		},
		"SERVER issue resolved": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-resolved.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **updated** bug [TES-4: Unit test summary 1](http://some-instance-test.centralus.cloudapp.azure.com:8080/browse/TES-4)",
			ExpectedFields: []*model.SlackAttachmentField{
				{
					Title: "",
					Value: "**Status:** ~~In Progress~~ Resolved",
					Short: false,
				},
				{
					Title: "",
					Value: "**Resolved:** ~~Open~~ Done",
					Short: false,
				},
			},
			CurrentInstance: true,
		},
		"SERVER issue reopened": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-reopened.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **updated** bug [TES-4: Unit test summary 1](http://some-instance-test.centralus.cloudapp.azure.com:8080/browse/TES-4)",
			ExpectedFields: []*model.SlackAttachmentField{
				{
					Title: "",
					Value: "**Reopened:** ~~Done~~ Open",
					Short: false,
				},
				{
					Title: "",
					Value: "**Status:** ~~Resolved~~ Reopened",
					Short: false,
				},
			},
			CurrentInstance: true,
		},
		"SERVER issue in progress": {
			Request:          testWebhookRequest("webhook-server-issue-updated-in-progress.json"),
			ExpectedHeadline: "Test User **updated** status from \"Reopened\" to \"In Progress\" on bug [TES-4: Unit test summary 1](http://some-instance-test.centralus.cloudapp.azure.com:8080/browse/TES-4)",
		},
		"SERVER issue closed": {
			Request:          testWebhookRequest("webhook-server-issue-updated-closed.json"),
			ExpectedHeadline: "Test User **updated** status from \"Resolved\" to \"Closed\" on bug [TES-4: Unit test summary 1](http://some-instance-test.centralus.cloudapp.azure.com:8080/browse/TES-4)",

			CurrentInstance: true,
		},
		"issue sprint": {
			Request:          testWebhookRequest("webhook-issue-updated-sprint.json"),
			ExpectedHeadline: "Test User **updated** Sprint from \"Sprint 1\" to \"Sprint 2\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"issue started working": {
			Request:          testWebhookRequest("webhook-issue-updated-started-working.json"),
			ExpectedHeadline: "Test User **updated** status from \"To Do\" to \"In Progress\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  true,
		},
		"CLOUD comment created": {
			Request:                 testWebhookRequest("webhook-cloud-comment-created.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **commented** on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "> Added a comment",
			CurrentInstance:         true,
		},
		"CLOUD comment updated": {
			Request:                 testWebhookRequest("webhook-cloud-comment-updated.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **edited comment** in story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "> Added a comment, then edited it",
			CurrentInstance:         true,
		},
		"CLOUD comment deleted": {
			Request:          testWebhookRequest("webhook-cloud-comment-deleted.json"),
			ExpectedHeadline: "Test User **deleted comment** in task [KT-7: s](https://mmtest.atlassian.net/browse/KT-7)",
			CurrentInstance:  true,
		},
		"SERVER issue commented": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-commented-1.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk **commented** on story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "> unik",
			CurrentInstance:         true,
		},
		"SERVER issue comment created indentation": {
			Request:         testWebhookRequest("webhook-issue-comment-created-indentation.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline: "User **commented** on story [TEST-4: unit testing](http://localhost:8082/browse/TEST-4)",
			ExpectedText:     "> [~Test] creating a test comment\r\n> \r\n> a second line for the test comment",
			CurrentInstance:  true,
		},
		"SERVER (old version) issue commented (no issue_event_type_name)": {
			Request:                 testWebhookRequest("webhook-server-old-issue-updated-no-event-type-commented.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk **commented** on story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "> unik",
			CurrentInstance:         true,
		},
		"SERVER issue comment deleted": {
			Request:          testWebhookRequest("webhook-server-issue-updated-comment-deleted.json"),
			ExpectedHeadline: "Lev Brouk **deleted comment** in story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:     "",
			CurrentInstance:  true,
		},
		"SERVER (old version) issue comment deleted (no issue_event_type_name)": {
			Request:         testWebhookRequest("webhook-server-old-issue-updated-no-event-type-comment-deleted.json"),
			ExpectedIgnored: true,
			ExpectedStatus:  http.StatusBadRequest,
			CurrentInstance: true,
		},
		"SERVER issue comment edited": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-comment-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk **edited comment** in story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "> and higher eeven higher",
			CurrentInstance:         true,
		},
		"SERVER (old version) issue comment edited (no issue_event_type_name)": {
			Request:                 testWebhookRequest("webhook-server-old-issue-updated-no-event-type-comment-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk **edited comment** in story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "> and higher eeven higher",
			CurrentInstance:         true,
		},
		"SERVER issue commented notify": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-commented-2.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **commented** on improvement [PRJA-42: test for notifications](http://test-server.azure.com:8080/browse/PRJA-42)",
			ExpectedText:            "> This is a test comment. We should act on it right away.",
			CurrentInstance:         true,
		},
		"SERVER: ignored comment created": {
			Request:         testWebhookRequest("webhook-server-comment-created.json"),
			ExpectedIgnored: true,
			CurrentInstance: true,
		},
		"SERVER: ignored comment updated": {
			Request:         testWebhookRequest("webhook-server-comment-updated.json"),
			ExpectedIgnored: true,
			CurrentInstance: true,
		},
		"SERVER: ignored comment deleted": {
			Request:         testWebhookRequest("webhook-server-comment-deleted.json"),
			ExpectedIgnored: true,
			CurrentInstance: true,
		},
		"issue created - no Instance": {
			Request:                 testWebhookRequest("webhook-issue-created.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long",
			ExpectedFields: []*model.SlackAttachmentField{
				&model.SlackAttachmentField{
					Title: "Priority",
					Value: "High",
					Short: true,
				},
			},
			CurrentInstance: false,
		},
		"issue created no fields - no Instance": {
			Request:                 testWebhookRequest("webhook-issue-created-no-relevant-fields.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long",
			ExpectedFields:          []*model.SlackAttachmentField{},
			CurrentInstance:         false,
		},
		"issue created no description - no Instance": {
			Request:                 testWebhookRequest("webhook-issue-created-no-description.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			// ExpectedText:            "Unit test description, not that long",
			ExpectedFields: []*model.SlackAttachmentField{
				&model.SlackAttachmentField{
					Title: "Priority",
					Value: "High",
					Short: true,
				},
			},
			CurrentInstance: false,
		},
		"issue created no description nor fields - no Instance": {
			Request:          testWebhookRequest("webhook-issue-created-no-description-nor-relevant-fields.json"),
			ExpectedStatus:   http.StatusOK,
			ExpectedHeadline: "Test User **created** story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"issue edited - no Instance": {
			Request:                 testWebhookRequest("webhook-issue-updated-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **edited** the description of story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long, a little longer now",
			CurrentInstance:         false,
		},
		"issue renamed - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-renamed.json"),
			ExpectedHeadline: "Test User **updated** summary from \"Unit test summary\" to \"Unit test summary 1\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:     "",
			CurrentInstance:  false,
		},
		"issue assigned nobody - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned-nobody.json"),
			ExpectedHeadline: "Test User **assigned** _nobody_ to story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"issue assigned - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned.json"),
			ExpectedHeadline: "Test User **assigned** Test User to story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"issue assigned on server - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned-on-server.json"),
			ExpectedHeadline: "Test User **assigned** Test User to improvement [PRJA-37: test](http://some-instance-test.centralus.cloudapp.azure.com:8080/browse/PRJA-37)",
			CurrentInstance:  false,
		},
		"issue attachments - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-attachments.json"),
			ExpectedHeadline: "Test User **attached** [test.gif] to, **removed** attachments [test.json] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"issue fix version - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-fix-version.json"),
			ExpectedHeadline: `Test User **updated** Fix Version from "v1" to "v2" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  false,
		},
		"issue issue type - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-issue-type.json"),
			ExpectedHeadline: `Test User **updated** issuetype from "Task" to "Bug" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  false,
		},
		"issue labels - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-labels.json"),
			ExpectedHeadline: "Test User **added** labels [sad] to, **removed** labels [bad] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"issue lowered priority - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-lowered-priority.json"),
			ExpectedHeadline: `Test User **updated** priority from "High" to "Low" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  false,
		},
		"issue raised priority - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-raised-priority.json"),
			ExpectedHeadline: `Test User **updated** priority from "Low" to "High" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
			CurrentInstance:  false,
		},
		"issue rank - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-rank.json"),
			ExpectedHeadline: "Test User **updated** Rank from \"~~none~~\" to \"ranked higher\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"issue reopened - no Instance": {
			Request:                 testWebhookRequest("webhook-issue-updated-reopened.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **updated** story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedFields: []*model.SlackAttachmentField{
				{
					Title: "",
					Value: "**Reopened:** ~~Done~~ Open",
					Short: false,
				},
				{
					Title: "",
					Value: "**Status:** ~~Done~~ To Do",
					Short: false,
				},
			},
			CurrentInstance: false,
		},
		"issue resolved - no Instance": {
			Request:                 testWebhookRequest("webhook-issue-updated-resolved.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **updated** story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedFields: []*model.SlackAttachmentField{
				{
					Title: "",
					Value: "**Resolved:** ~~Open~~ Done",
					Short: false,
				},
				{
					Title: "",
					Value: "**Status:** ~~In Progress~~ Done",
					Short: false,
				},
			},
			CurrentInstance: false,
		},
		"issue sprint - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-sprint.json"),
			ExpectedHeadline: "Test User **updated** Sprint from \"Sprint 1\" to \"Sprint 2\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"issue started working - no Instance": {
			Request:          testWebhookRequest("webhook-issue-updated-started-working.json"),
			ExpectedHeadline: "Test User **updated** status from \"To Do\" to \"In Progress\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			CurrentInstance:  false,
		},
		"CLOUD comment created - no Instance": {
			Request:                 testWebhookRequest("webhook-cloud-comment-created.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **commented** on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "> Added a comment",
			CurrentInstance:         false,
		},
		"CLOUD comment updated - no Instance": {
			Request:                 testWebhookRequest("webhook-cloud-comment-updated.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **edited comment** in story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "> Added a comment, then edited it",
			CurrentInstance:         false,
		},
		"CLOUD comment deleted - no Instance": {
			Request:          testWebhookRequest("webhook-cloud-comment-deleted.json"),
			ExpectedHeadline: "Test User **deleted comment** in task [KT-7: s](https://mmtest.atlassian.net/browse/KT-7)",
			CurrentInstance:  false,
		},
		"SERVER issue commented - no Instance": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-commented-1.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk **commented** on story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "> unik",
			CurrentInstance:         false,
		},
		"SERVER issue comment deleted - no Instance": {
			Request:          testWebhookRequest("webhook-server-issue-updated-comment-deleted.json"),
			ExpectedHeadline: "Lev Brouk **deleted comment** in story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:     "",
			CurrentInstance:  false,
		},
		"SERVER issue comment edited - no Instance": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-comment-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk **edited comment** in story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "> and higher eeven higher",
			CurrentInstance:         false,
		},
		"SERVER issue commented notify - no Instance": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-commented-2.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User **commented** on improvement [PRJA-42: test for notifications](http://test-server.azure.com:8080/browse/PRJA-42)",
			ExpectedText:            "> This is a test comment. We should act on it right away.",
			CurrentInstance:         false,
		},
		"SERVER: ignored comment created - no Instance": {
			Request:         testWebhookRequest("webhook-server-comment-created.json"),
			ExpectedIgnored: true,
			CurrentInstance: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			api.On("LogDebug",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogError",
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string"),
				mock.AnythingOfTypeArgument("string")).Return(nil)

			api.On("GetUserByUsername", "theuser").Return(&model.User{
				Id: "theuserid",
			}, (*model.AppError)(nil))
			api.On("GetChannelByNameForTeamName", "theteam", "thechannel",
				false).Run(func(args mock.Arguments) {
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, (*model.AppError)(nil))
			}).Return(&model.Channel{
				Id:     "thechannelid",
				TeamId: "theteamid",
			}, (*model.AppError)(nil))

			p := Plugin{}
			p.updateConfig(func(conf *config) {
				conf.Secret = validConfiguration.Secret
			})
			p.SetAPI(api)

			p.userStore = mockUserStore{}
			p.instanceStore = p.getMockInstanceStoreKV(1)

			w := httptest.NewRecorder()
			recorder := &testWebhookWrapper{}
			prev := webhookWrapperFunc
			defer func() { webhookWrapperFunc = prev }()
			webhookWrapperFunc = func(wh Webhook) Webhook {
				recorder.Webhook = wh
				return recorder
			}
			p.ServeHTTP(&plugin.Context{}, w, tc.Request)
			expectedStatus := http.StatusOK
			if tc.ExpectedStatus != 0 {
				expectedStatus = tc.ExpectedStatus
			}
			assert.Equal(t, expectedStatus, w.Result().StatusCode)

			if tc.ExpectedIgnored {
				require.Nil(t, recorder.postedToChannel)
				return
			}

			require.NotNil(t, recorder.postedToChannel)
			post := recorder.postedToChannel

			if !tc.ExpectedSlackAttachment {
				assert.Equal(t, tc.ExpectedHeadline, post.Message)
				return
			}

			require.NotNil(t, post.Props)
			require.NotNil(t, post.Props["attachments"])
			attachments := post.Props["attachments"].([]*model.SlackAttachment)
			require.Equal(t, 1, len(attachments))

			sa := attachments[0]
			assert.Equal(t, tc.ExpectedHeadline, sa.Pretext)
			assert.Equal(t, tc.ExpectedText, sa.Text)
			require.Equal(t, len(tc.ExpectedFields), len(sa.Fields))
			for i := range tc.ExpectedFields {
				assert.Equal(t, tc.ExpectedFields[i].Title, sa.Fields[i].Title)
				assert.Equal(t, tc.ExpectedFields[i].Value, sa.Fields[i].Value)
				assert.Equal(t, tc.ExpectedFields[i].Short, sa.Fields[i].Short)
			}
		})
	}
}
