// Torq Private Channel Sync
// version 0.5.0

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"
)

// Plugin implements the Mattermost server plugin interface. Embedding
// plugin.MattermostPlugin wires up p.API automatically just before OnActivate.
type Plugin struct {
	plugin.MattermostPlugin

	confStore  configStore
	torqClient *torqClient
}

func (p *Plugin) OnActivate() error {
	if err := p.OnConfigurationChange(); err != nil {
		return fmt.Errorf("failed to load configuration on activate: %w", err)
	}

	p.torqClient = newTorqClient(&http.Client{
		Timeout: 10 * time.Second,
	}, p)

	p.API.LogInfo("Torq Sync plugin activated")
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.API.LogInfo("Torq Sync plugin deactivated")
	return nil
}
