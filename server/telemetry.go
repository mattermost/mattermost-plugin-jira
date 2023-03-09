package main

func (p *Plugin) TrackEvent(event string, properties map[string]interface{}) {
	err := p.tracker.TrackEvent(event, properties)
	if err != nil {
		p.API.LogDebug("Error sending telemetry event", "event", event, "error", err.Error())
	}
}

func (p *Plugin) TrackUserEvent(event, userID string, properties map[string]interface{}) {
	err := p.tracker.TrackUserEvent(event, userID, properties)
	if err != nil {
		p.API.LogDebug("Error sending user telemetry event", "event", event, "error", err.Error())
	}
}

func (p *Plugin) OnSendDailyTelemetry() {
	args := map[string]interface{}{}

	// Jira instances
	server, cloud := 0, 0
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		p.API.LogWarn("Failed to get instances for telemetry", "error", err)
	} else {
		for _, id := range instances.IDs() {
			switch instances.Get(id).Type {
			case ServerInstanceType:
				server++
			case CloudInstanceType:
				cloud++
			}
		}
		args["instance_count"] = server + cloud
		if server > 0 {
			args["server_instance_count"] = server
		}
		if cloud > 0 {
			args["cloud_instance_count"] = cloud
		}

		// Subscriptions
		numSubscriptions := 0
		var subs *Subscriptions
		for _, id := range instances.IDs() {
			subs, err = p.getSubscriptions(id)
			if err != nil {
				p.API.LogWarn("Failed to get subscriptions for telemetry", "error", err)
			}
			numSubscriptions += len(subs.Channel.ByID)
		}

		args["subscriptions"] = numSubscriptions
	}

	// Connected users
	connected, err := p.userStore.CountUsers()
	if err != nil {
		p.API.LogWarn("Failed to get the number of connected users for telemetry", "error", err)
	} else {
		args["connected_user_count"] = connected
	}

	p.TrackEvent("stats", args)
}
