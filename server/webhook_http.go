// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"fmt"
	"math"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
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
	eventUpdatedIssuetype
)

const maskLegacy = eventCreated |
	eventUpdatedReopened |
	eventUpdatedResolved |
	eventDeletedUnresolved

const maskComments = eventCreatedComment |
	eventDeletedComment |
	eventUpdatedComment

const maskDefault = maskLegacy |
	eventUpdatedAssignee

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
	"updated_comments":    maskComments,            // comment events
	"updated_all":         maskAll,                 // all events
}

var ErrWebhookIgnored = errors.New("Webhook purposely ignored")

func httpWebhook(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	// Validate the request and extract params
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed,
			fmt.Errorf("Request: " + r.Method + " is not allowed, must be POST")
	}
	cfg := p.getConfig()
	if cfg.Secret == "" {
		return http.StatusForbidden, fmt.Errorf("Jira plugin not configured correctly; must provide Secret")
	}
	secret := r.FormValue("secret")
	// secret may be URL-escaped, potentially mroe than once. Loop until there
	// are no % escapes left.
	for {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(cfg.Secret)) == 1 {
			break
		}

		unescaped, _ := url.QueryUnescape(secret)
		if unescaped == secret {
			return http.StatusForbidden,
				errors.New("Request URL: secret did not match")
		}
		secret = unescaped
	}
	teamName := r.FormValue("team")
	if teamName == "" {
		return http.StatusBadRequest,
			errors.New("Request URL: no team name found")
	}
	channelName := r.FormValue("channel")
	if channelName == "" {
		return http.StatusBadRequest,
			errors.New("Request URL: no channel name found")
	}
	eventMask := maskDefault
	for key, paramMask := range eventParamMasks {
		if r.FormValue(key) == "" {
			continue
		}
		eventMask = eventMask | paramMask
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelName, false)
	if appErr != nil {
		return appErr.StatusCode, appErr
	}

	wh, _, err := ParseWebhook(r.Body)
	if err == ErrWebhookIgnored {
		return http.StatusOK, nil
	}
	if err != nil {
		return http.StatusBadRequest, err
	}

	// Attempt to send webhook notifications to connected users.
	_, statusCode, err := wh.PostNotifications(p)
	if err != nil {
		return statusCode, err
	}

	// Send webhook events to subscribed channels. This will work even if there isn't an instance installed.
	// Skip events we don't need to post
	if eventMask&wh.EventMask() == 0 {
		return http.StatusOK, nil
	}

	// Post the event to the subscribed channel
	_, statusCode, err = wh.PostToChannel(p, channel.Id, p.getUserID())
	if err != nil {
		return statusCode, err
	}

	return http.StatusOK, nil
}
