package jiraTracker

import "github.com/mattermost/mattermost-plugin-jira/server/utils/telemetry"

const (
	userConnectedEvent    = "userConnected"
	userDisconnectedEvent = "userDisconnected"
)

type Tracker interface {
	TrackUserConnected(userID string)
	TrackUserDisconnected(userID string)
}

func New(t telemetry.Tracker) Tracker {
	return &tracker{
		tracker: t,
	}
}

type tracker struct {
	tracker telemetry.Tracker
}

func (t *tracker) TrackUserConnected(userID string) {
	t.tracker.TrackUserEvent(userConnectedEvent, userID, map[string]interface{}{})
}

func (t *tracker) TrackUserDisconnected(userID string) {
	t.tracker.TrackUserEvent(userDisconnectedEvent, userID, map[string]interface{}{})
}
