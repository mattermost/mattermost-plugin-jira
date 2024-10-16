package main

import (
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/telemetry"
)

func (p *Plugin) TrackEvent(event string, properties map[string]interface{}) {
	err := p.tracker.TrackEvent(event, properties)
	if err != nil {
		p.client.Log.Debug("Error sending telemetry event", "event", event, "error", err.Error())
	}
}

func (p *Plugin) TrackUserEvent(event, userID string, properties map[string]interface{}) {
	err := p.tracker.TrackUserEvent(event, userID, properties)
	if err != nil {
		p.client.Log.Debug("Error sending user telemetry event", "event", event, "error", err.Error())
	}
}

func (p *Plugin) OnSendDailyTelemetry() {
	args := map[string]interface{}{}

	// Jira instances
	server, cloud, err := p.instanceCount()
	if err != nil {
		p.client.Log.Warn("Failed to get instances for telemetry", "error", err)
	} else {
		args["instance_count"] = server + cloud
		if server > 0 {
			args["server_instance_count"] = server
		}
		if cloud > 0 {
			args["cloud_instance_count"] = cloud
		}
	}

	subscriptionCount, err := p.subscriptionCount()
	if err != nil {
		p.client.Log.Warn("Failed to get the number of subscriptions for telemetry", "error", err)
	} else {
		args["subscriptions"] = subscriptionCount
	}

	// Connected users
	connected, err := p.userCount()
	if err != nil {
		p.client.Log.Warn("Failed to get the number of connected users for telemetry", "error", err)
	} else {
		args["connected_user_count"] = connected
	}

	p.TrackEvent("stats", args)
}

func (p *Plugin) instanceCount() (server int, cloud int, err error) {
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return
	}

	for _, id := range instances.IDs() {
		switch instances.Get(id).Type {
		case ServerInstanceType:
			server++
		case CloudInstanceType:
			cloud++
		}
	}

	return
}

func (p *Plugin) subscriptionCount() (int, error) {
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return 0, err
	}

	// Subscriptions
	numSubscriptions := 0
	var subs *Subscriptions
	for _, id := range instances.IDs() {
		subs, err = p.getSubscriptions(id)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get subscription %q for telemetry", id)
		}
		numSubscriptions += len(subs.Channel.ByID)
	}

	return numSubscriptions, nil
}

func (p *Plugin) userCount() (int, error) {
	return p.userStore.CountUsers()
}

// Initialize telemetry setups the tracker/clients needed to send telemetry data.
// The telemetry.NewTrackerConfig(...) param will take care of extract/parse the config to set the right settings.
// If you don't want the default behavior you still can pass a different telemetry.TrackerConfig data.
func (p *Plugin) initializeTelemetry() {
	var err error

	// Telemetry client
	p.telemetryClient, err = telemetry.NewRudderClient()
	if err != nil {
		p.API.LogWarn("Telemetry client not started", "error", err.Error())
		return
	}

	// Get config values
	p.tracker = telemetry.NewTracker(
		p.telemetryClient,
		p.API.GetDiagnosticId(),
		p.API.GetServerVersion(),
		manifest.Id,
		manifest.Version,
		"jira",
		telemetry.NewTrackerConfig(p.API.GetConfig()),
		telemetry.NewLogger(p.API),
	)
}
