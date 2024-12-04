package main

import (
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type SupportPacket struct {
	Version string `yaml:"version"`

	InstanceCount       int `yaml:"instance_count"`
	ServerInstanceCount int `yaml:"server_instance_count"`
	CloudInstanceCount  int `yaml:"cloud_instance_count"`
	SubscriptionCount   int `yaml:"subscription_count"`
	ConnectedUserCount  int `yaml:"connected_user_count"`
}

func (p *Plugin) GenerateSupportData(_ *plugin.Context) ([]*model.FileData, error) {
	var result *multierror.Error

	connectedUserCount, err := p.userCount()
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "Failed to get the number of connected users for Support Packet"))
	}

	server, cloud, err := p.instanceCount()
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "Failed to get the number of instances for Support Packet"))
	}

	subscriptionCount, err := p.subscriptionCount()
	if err != nil {
		result = multierror.Append(result, errors.Wrap(err, "Failed to get the number of subscriptions for Support Packet"))
	}

	diagnostics := SupportPacket{
		Version:             manifest.Version,
		InstanceCount:       server + cloud,
		ServerInstanceCount: server,
		CloudInstanceCount:  cloud,
		SubscriptionCount:   subscriptionCount,
		ConnectedUserCount:  connectedUserCount,
	}
	b, err := yaml.Marshal(diagnostics)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal diagnostics")
	}

	return []*model.FileData{{
		Filename: filepath.Join(manifest.Id, "diagnostics.yaml"),
		Body:     b,
	}}, result.ErrorOrNil()
}
