// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"math"
	"net/http"
	"net/url"
)

const (
	PostTypeComment  = "custom_jira_comment"
	PostTypeMention  = "custom_jira_mention"
	PostTypeAssigned = "custom_jira_assigned"
)

const (
	eventCreated = uint64(1 << iota)
	eventCreatedComment
	eventDeleted
	eventDeletedComment
	eventDeletedUnresolved
	eventUpdatedAssignee
	eventUpdatedAttachment
	eventUpdatedComment
	eventUpdatedDescription
	eventUpdatedLabels
	eventUpdatedPriority
	eventUpdatedRank
	eventUpdatedReopened
	eventUpdatedResolved
	eventUpdatedSprint
	eventUpdatedStatus
	eventUpdatedSummary
)

const maskLegacy = eventCreated |
	eventUpdatedReopened |
	eventUpdatedResolved |
	eventDeletedUnresolved

const maskComments = eventCreatedComment |
	eventDeletedComment |
	eventUpdatedComment

const maskDefault = maskLegacy |
	eventUpdatedAssignee |
	maskComments

const maskAll = math.MaxUint64

// The keys listed here can be used in the Jira webhook URL to control what events
// are posted to Mattermost. A matching parameter with a non-empty value must
// be added to turn on the event display.
var eventParamMasks = map[string]uint64{
	"updated_attachment":  eventUpdatedAttachment,  // updated attachments
	"updated_description": eventUpdatedDescription, // issue description edited
	"updated_labels":      eventUpdatedLabels,      // updated labels
	"updated_prioity":     eventUpdatedPriority,    // changes in priority
	"updated_rank":        eventUpdatedRank,        // ranked higher or lower
	"updated_sprint":      eventUpdatedSprint,      // assigned to a different sprint
	"updated_status":      eventUpdatedStatus,      // transitions like Done, In Progress
	"updated_summary":     eventUpdatedSummary,     // issue renamed
	"updated_all":         maskAll,                 // all events
}

func httpWebhook(a *Action) error {
	if a.PluginConfig.Secret == "" || a.PluginConfig.UserName == "" {
		return a.RespondError(http.StatusForbidden, nil,
			"Jira plugin not configured correctly; must provide Secret and UserName")
	}
	secret := a.HTTPRequest.FormValue("secret")
	// secret may be URL-escaped, potentially mroe than once. Loop until there
	// are no % escapes left.
	for {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(a.PluginConfig.Secret)) == 1 {
			break
		}

		unescaped, _ := url.QueryUnescape(secret)
		if unescaped == secret {
			return a.RespondError(http.StatusForbidden, nil, "Request URL: secret did not match")
		}
		secret = unescaped
	}
	teamName := a.HTTPRequest.FormValue("team")
	if teamName == "" {
		return a.RespondError(http.StatusBadRequest, nil, "Request URL: no team name found")
	}
	channelName := a.HTTPRequest.FormValue("channel")
	if channelName == "" {
		return a.RespondError(http.StatusBadRequest, nil, "Request URL: no channel name found")
	}
	eventMask := maskDefault
	for key, paramMask := range eventParamMasks {
		if a.HTTPRequest.FormValue(key) == "" {
			continue
		}
		eventMask = eventMask | paramMask
	}

	botUser, appErr := a.API.GetUserByUsername(a.PluginConfig.UserName)
	if appErr != nil {
		return a.RespondError(appErr.StatusCode, appErr)
	}
	channel, appErr := a.API.GetChannelByNameForTeamName(teamName, channelName, false)
	if appErr != nil {
		return a.RespondError(appErr.StatusCode, appErr)
	}

	wh, _, err := ParseWebhook(a.HTTPRequest.Body)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err)
	}

	wh.PostNotifications(a.PluginConfig, a.API, a.UserStore, a.Instance)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	// Skip events we don't need to post
	if eventMask&wh.EventMask() == 0 {
		return nil
	}

	// Post the event to the subscribed channel
	_, statusCode, err := wh.PostToChannel(a.API, channel.Id, botUser.Id)
	if err != nil {
		return a.RespondError(statusCode, err)
	}

	return nil
}
