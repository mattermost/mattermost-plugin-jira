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

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
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

func (wh testWebhookWrapper) EventMask() uint64 {
	return wh.Webhook.EventMask()
}
func (wh *testWebhookWrapper) PostToChannel(p *Plugin, channelId, fromUserId string) (*model.Post, int, error) {
	post, status, err := wh.Webhook.PostToChannel(p, channelId, fromUserId)
	if post != nil {
		wh.postedToChannel = post
	}
	return post, status, err
}
func (wh *testWebhookWrapper) PostNotifications(p *Plugin) ([]*model.Post, int, error) {
	posts, status, err := wh.Webhook.PostNotifications(p)
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
		ExpectedIgnored         bool
		CurrentInstance         bool
	}{
		"issue created": {
			Request:                 testWebhookRequest("webhook-issue-created.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User created story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
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
			ExpectedHeadline:        "Test User created story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long",
			ExpectedFields:          []*model.SlackAttachmentField{},
		},
		"issue created no description": {
			Request:                 testWebhookRequest("webhook-issue-created-no-description.json"),
			ExpectedStatus:          http.StatusOK,
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User created story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
			// ExpectedText:            "Unit test description, not that long",
			ExpectedFields: []*model.SlackAttachmentField{
				&model.SlackAttachmentField{
					Title: "Priority",
					Value: "High",
					Short: true,
				},
			},
		},
		"issue created no description nor fields": {
			Request:          testWebhookRequest("webhook-issue-created-no-description-nor-relevant-fields.json"),
			ExpectedStatus:   http.StatusOK,
			ExpectedHeadline: "Test User created story [TES-41: Unit test summary](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue edited": {
			Request:                 testWebhookRequest("webhook-issue-updated-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User edited the description of story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Unit test description, not that long, a little longer now",
		},
		"issue renamed": {
			Request:          testWebhookRequest("webhook-issue-updated-renamed.json"),
			ExpectedHeadline: "Test User renamed story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:     "",
		},
		"issue assigned nobody": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned-nobody.json"),
			ExpectedHeadline: "Test User assigned _nobody_ to story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue assigned": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned.json"),
			ExpectedHeadline: "Test User assigned Test User to story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue assigned on server": {
			Request:          testWebhookRequest("webhook-issue-updated-assigned-on-server.json"),
			ExpectedHeadline: "Test User assigned Test User to improvement [PRJA-37: test](http://some-instance-test.centralus.cloudapp.azure.com:8080/browse/PRJA-37)",
		},
		"issue attachments": {
			Request:          testWebhookRequest("webhook-issue-updated-attachments.json"),
			ExpectedHeadline: "Test User attached [test.gif] to, removed attachments [test.json] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue labels": {
			Request:          testWebhookRequest("webhook-issue-updated-labels.json"),
			ExpectedHeadline: "Test User added labels [sad] to, removed labels [bad] from story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue lowered priority": {
			Request:          testWebhookRequest("webhook-issue-updated-lowered-priority.json"),
			ExpectedHeadline: `Test User updated priority from "High" to "Low" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
		},
		"issue raised priority": {
			Request:          testWebhookRequest("webhook-issue-updated-raised-priority.json"),
			ExpectedHeadline: `Test User updated priority from "Low" to "High" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)`,
		},
		"issue rank": {
			Request:          testWebhookRequest("webhook-issue-updated-rank.json"),
			ExpectedHeadline: "Test User ranked higher story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue reopened": {
			Request:          testWebhookRequest("webhook-issue-updated-reopened.json"),
			ExpectedHeadline: "Test User reopened story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue resolved": {
			Request:          testWebhookRequest("webhook-issue-updated-resolved.json"),
			ExpectedHeadline: "Test User resolved story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"issue sprint": {
			Request:          testWebhookRequest("webhook-issue-updated-sprint.json"),
			ExpectedHeadline: "Test User moved story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41) to Sprint 2",
		},
		"issue started working": {
			Request:          testWebhookRequest("webhook-issue-updated-started-working.json"),
			ExpectedHeadline: "Test User updated status from \"To Do\" to \"In Progress\" on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"CLOUD comment created": {
			Request:                 testWebhookRequest("webhook-cloud-comment-created.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User commented on story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Added a comment",
		},
		"CLOUD comment updated": {
			Request:                 testWebhookRequest("webhook-cloud-comment-updated.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User edited comment in story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
			ExpectedText:            "Added a comment, then edited it",
		},
		"CLOUD comment deleted": {
			Request:          testWebhookRequest("webhook-cloud-comment-deleted.json"),
			ExpectedHeadline: "Test User deleted comment in story [TES-41: Unit test summary 1](https://some-instance-test.atlassian.net/browse/TES-41)",
		},
		"SERVER issue commented": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-commented-1.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk commented on story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "unik",
		},
		"SERVER issue comment deleted": {
			Request:          testWebhookRequest("webhook-server-issue-updated-comment-deleted.json"),
			ExpectedHeadline: "Lev Brouk deleted comment in story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:     "",
		},
		"SERVER issue comment edited": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-comment-edited.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Lev Brouk edited comment in story [PRJX-14: As a user, I can find important items on the board by using the customisable ...](http://sales-jira.centralus.cloudapp.azure.com:8080/browse/PRJX-14)",
			ExpectedText:            "and higher eeven higher",
		},
		"SERVER issue commented notify": {
			Request:                 testWebhookRequest("webhook-server-issue-updated-commented-2.json"),
			ExpectedSlackAttachment: true,
			ExpectedHeadline:        "Test User commented on improvement [PRJA-42: test for notifications](http://test-server.azure.com:8080/browse/PRJA-42)",
			ExpectedText:            "This is a test comment. We should act on it right away.",
		},
		"SERVER: ignored comment created": {
			Request:         testWebhookRequest("webhook-server-comment-created.json"),
			ExpectedIgnored: true,
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

			if tc.CurrentInstance {
				p.currentInstanceStore = mockCurrentInstanceStore{&p}
			} else {
				p.currentInstanceStore = mockCurrentInstanceStoreNoInstance{&p}
			}

			p.userStore = mockUserStore{}

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
