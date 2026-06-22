package main

import (
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/plugin"
)

// configuration captures the plugin's runtime settings, sourced from plugin.json's
// settings_schema via OnConfigurationChange.
type configuration struct {
	TorqWebhookURL         string
	TorqWebhookSecretKey   string
	TorqWebhookSecret      string
	SyncAllPrivateChannels bool
	ChannelAllowList       string
	IncludeMessageContent  bool

	// derived
	allowedChannelIDs map[string]bool
}

func (c *configuration) Clone() *configuration {
	clone := *c
	if c.allowedChannelIDs != nil {
		clone.allowedChannelIDs = make(map[string]bool, len(c.allowedChannelIDs))
		for k, v := range c.allowedChannelIDs {
			clone.allowedChannelIDs[k] = v
		}
	}
	return &clone
}

func (c *configuration) buildDerived() {
	c.allowedChannelIDs = map[string]bool{}
	for _, id := range strings.Split(c.ChannelAllowList, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			c.allowedChannelIDs[id] = true
		}
	}
}

func (c *configuration) channelAllowed(channelID string) bool {
	if c.SyncAllPrivateChannels {
		return true
	}
	return c.allowedChannelIDs[channelID]
}

// Plugin-wide config holder with safe concurrent access. Hooks fire concurrently
// from multiple goroutines, so reads/writes to configuration must be guarded.
type configStore struct {
	mu  sync.RWMutex
	cfg *configuration
}

func (s *configStore) get() *configuration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *configStore) set(cfg *configuration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func (p *Plugin) getConfiguration() *configuration {
	return p.confStore.get()
}

// OnConfigurationChange is invoked by the server whenever an admin saves settings
// in the System Console for this plugin.
func (p *Plugin) OnConfigurationChange() error {
	var cfg configuration
	if err := p.API.LoadPluginConfiguration(&cfg); err != nil {
		return err
	}
	cfg.buildDerived()

	if cfg.TorqWebhookURL == "" {
		p.API.LogWarn("Torq Sync: TorqWebhookURL is not configured; events will not be forwarded")
	}

	p.confStore.set(&cfg)
	return nil
}

var _ plugin.MattermostPlugin // keep import used if embedding changes later
