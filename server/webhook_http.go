// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/subtle"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

const (
	PostTypeComment  = "custom_jira_comment"
	PostTypeMention  = "custom_jira_mention"
	PostTypeAssigned = "custom_jira_assigned"
)

// The keys listed here can be used in the Jira webhook URL to control what events
// are posted to Mattermost. A matching parameter with a non-empty value must
// be added to turn on the event display.
var eventParamMasks = map[string]StringSet{
	"updated_attachment":  NewStringSet(eventUpdatedAttachment),  // updated attachments
	"updated_description": NewStringSet(eventUpdatedDescription), // issue description edited
	"updated_labels":      NewStringSet(eventUpdatedLabels),      // updated labels
	"updated_prioity":     NewStringSet(eventUpdatedPriority),    // changes in priority
	"updated_rank":        NewStringSet(eventUpdatedRank),        // ranked higher or lower
	"updated_sprint":      NewStringSet(eventUpdatedSprint),      // assigned to a different sprint
	"updated_status":      NewStringSet(eventUpdatedStatus),      // transitions like Done, In Progress
	"updated_summary":     NewStringSet(eventUpdatedSummary),     // issue renamed
	"updated_comments":    commentEvents,                         // comment events
	"updated_all":         allEvents,                             // all events
}

var ErrWebhookIgnored = errors.New("Webhook purposely ignored")

func (p *Plugin) httpWebhook(w http.ResponseWriter, r *http.Request, instanceID types.ID) (status int, err error) {
	conf := p.getConfig()
	start := time.Now()
	size := utils.ByteSize(0)
	defer func() {
		isError, isIgnored := false, false
		switch err {
		case nil:
			break
		case ErrWebhookIgnored:
			// ignore ErrWebhookIgnored - from here up it's a success
			isIgnored = true
			err = nil
		default:
			isError = true
		}
		if conf.stats != nil {
			conf.stats.EnsureEndpoint("jira/webhook/response").Record(size, 0, time.Since(start), isError, isIgnored)
		}
	}()

	// Validate the request and extract params
	if r.Method != http.MethodPost {
		return respondErr(w, http.StatusMethodNotAllowed,
			fmt.Errorf("Request: "+r.Method+" is not allowed, must be POST"))
	}
	if conf.Secret == "" {
		return respondErr(w, http.StatusForbidden,
			fmt.Errorf("JIRA plugin not configured correctly; must provide Secret"))
	}
	status, err = verifyHTTPSecret(conf.Secret, r.FormValue("secret"))
	if err != nil {
		return respondErr(w, status, err)
	}
	teamName := r.FormValue("team")
	if teamName == "" {
		return respondErr(w, http.StatusBadRequest,
			errors.New("Request URL: no team name found"))
	}
	channelName := r.FormValue("channel")
	if channelName == "" {
		return respondErr(w, http.StatusBadRequest,
			errors.New("Request URL: no channel name found"))
	}

	selectedEvents := defaultEvents.Add()
	for key, paramMask := range eventParamMasks {
		if r.FormValue(key) == "" {
			continue
		}
		selectedEvents = selectedEvents.Union(paramMask)
	}

	bb, err := ioutil.ReadAll(r.Body)
	size = utils.ByteSize(len(bb))
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}

	channel, appErr := p.API.GetChannelByNameForTeamName(teamName, channelName, false)
	if appErr != nil {
		return respondErr(w, appErr.StatusCode, appErr)
	}

	wh, err := ParseWebhook(bb)
	if err == ErrWebhookIgnored {
		return respondErr(w, http.StatusOK, err)
	}
	if err != nil {
		return respondErr(w, http.StatusBadRequest, err)
	}

	// Skip events we don't need to post
	if selectedEvents.Intersection(wh.Events()).Len() == 0 {
		return http.StatusOK, nil
	}

	// Post the event to the channel
	_, statusCode, err := wh.PostToChannel(p, instanceID, channel.Id, p.getUserID())
	if err != nil {
		return respondErr(w, statusCode, err)
	}

	return http.StatusOK, nil
}

func verifyHTTPSecret(expected, got string) (status int, err error) {
	for {
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1 {
			break
		}

		unescaped, _ := url.QueryUnescape(got)
		if unescaped == got {
			return http.StatusForbidden,
				errors.New("Request URL: secret did not match")
		}
		got = unescaped
	}

	return 0, nil
}
