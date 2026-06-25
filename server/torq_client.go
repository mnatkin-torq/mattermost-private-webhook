// Torq Private Channel Sync
// version 0.5.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// torqClient sends event payloads to a Torq webhook trigger. All sends happen
// on a background goroutine so a slow or unreachable Torq endpoint can never
// add latency to the Mattermost hook that triggered it (MessageHasBeenPosted etc.
// run inline with the user-facing request in some code paths).
type torqClient struct {
	httpClient *http.Client
	plugin     *Plugin
}

func newTorqClient(httpClient *http.Client, p *Plugin) *torqClient {
	return &torqClient{httpClient: httpClient, plugin: p}
}

// eventEnvelope is the JSON shape POSTed to Torq. Keep this stable -- Torq's
// HTTP trigger step and any downstream parsing logic will key off these field
// names. Add new optional fields rather than renaming existing ones.
type eventEnvelope struct {
	EventType string `json:"event_type"`
	Timestamp int64  `json:"timestamp_ms"`

	TeamID    string `json:"team_id,omitempty"`
	ChannelID string `json:"channel_id"`

	UserID string `json:"user_id,omitempty"`

	PostID  string `json:"post_id,omitempty"`
	Message string `json:"message,omitempty"`

	// Membership-change events
	TargetUserID string `json:"target_user_id,omitempty"`

	Extra map[string]any `json:"extra,omitempty"`
}

// send fires the event at Torq asynchronously with retries. Failures are logged
// but never returned to the caller -- a Torq outage must not affect Mattermost.
func (c *torqClient) send(evt eventEnvelope) {
	cfg := c.plugin.getConfiguration()
	if cfg.TorqWebhookURL == "" {
		return
	}

	go func() {
		const maxAttempts = 4
		backoff := 500 * time.Millisecond

		body, err := json.Marshal(evt)
		if err != nil {
			c.plugin.API.LogError("Torq Sync: failed to marshal event", "err", err.Error())
			return
		}

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := c.post(ctx, cfg.TorqWebhookURL, cfg.TorqWebhookSecretKey, cfg.TorqWebhookSecret, body)
			cancel()

			if err == nil {
				return
			}

			c.plugin.API.LogWarn("Torq Sync: failed to deliver event to Torq",
				"attempt", attempt,
				"event_type", evt.EventType,
				"channel_id", evt.ChannelID,
				"err", err.Error(),
			)

			if attempt < maxAttempts {
				time.Sleep(backoff)
				backoff *= 2
			}
		}

		c.plugin.API.LogError("Torq Sync: exhausted retries delivering event to Torq",
			"event_type", evt.EventType,
			"channel_id", evt.ChannelID,
		)
	}()
}

func (c *torqClient) post(ctx context.Context, url, secretkey string, secret string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if secretkey != "" && secret != ""  {
		req.Header.Set(secretkey, secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return nil
}
