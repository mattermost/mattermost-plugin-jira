package telemetry

import "github.com/mattermost/mattermost-server/v5/plugin"

type Tracker interface {
	Track(event string, properties map[string]interface{})
	TrackUserEvent(event string, userID string, properties map[string]interface{})
}

type Client interface {
	Enqueue(t Track) error
	Close() error
}

type Track struct {
	UserID     string
	Event      string
	Properties map[string]interface{}
}

type tracker struct {
	client        Client
	diagnosticID  string
	serverVersion string
	pluginID      string
	pluginVersion string
	enabled       bool
}

func NewTracker(c Client, diagnosticID, serverVersion, pluginID, pluginVersion string, enableDiagnostics bool) Tracker {
	return &tracker{
		client:        c,
		diagnosticID:  diagnosticID,
		serverVersion: serverVersion,
		pluginID:      pluginID,
		pluginVersion: pluginVersion,
		enabled:       enableDiagnostics,
	}
}

func (t *tracker) Track(event string, properties map[string]interface{}) {
	if !t.enabled || t.client == nil {
		return
	}

	event = t.pluginID + "-" + event
	properties["PluginID"] = t.pluginID
	properties["PluginVersion"] = t.pluginVersion
	properties["ServerVersion"] = t.serverVersion

	var p *plugin.MattermostPlugin
	err := t.client.Enqueue(Track{
		UserID:     t.diagnosticID,
		Event:      event,
		Properties: properties,
	})
	if err != nil {
		p.API.LogWarn("cannot enqueue telemetry event, err=%s", err.Error())
	}
}

func (t *tracker) TrackUserEvent(event string, userID string, properties map[string]interface{}) {
	properties["UserActualID"] = userID
	t.Track(event, properties)
}
