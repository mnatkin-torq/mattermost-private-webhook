# Torq Mattermost Private Webhook PlugIn

## PURPOSE
Mattermost's classic Outgoing Webhooks feature is scoped to a specific channel (or all channels in a team for public ones), but it's designed around trigger-word matching on messages in public channels only. There's no admin console option to attach an outgoing webhook to a private channel — it's a deliberate product limitation, not a config you're missing.
So for private channel data, you're left with two real options: the websocket, or Mattermost's Plugin system, which can run server-side and isn't bound by that public-channel restriction.

Since Torq doesn't hold a persistent inbound connection (Torq does not support websockets), we need something on the Mattermost side (or adjacent to it) that does, and which then talks to Torq over native HTTP request/response — which Torq handles natively via its webhook trigger.
Mattermost supports server plugins that hook directly into the server's internal event system — not the websocket, the actual Go-level Hooks interface (MessageHasBeenPosted, ChannelHasBeenCreated, UserHasJoinedChannel, etc.). This runs inside the Mattermost server process, sees private channel events natively (no membership/visibility issue), and can make an outbound HTTPS POST to a Torq webhook URL per event.

No persistent connection problem at all — it's an in-process hook, not a socket.
This requires System Admin access to install, and someone to maintain a small Go plugin (this feature).

## Building and deploying (if starting from source)

### System Requirements
This PlugIn requires a Mattermost minimum server version of 7.8.0

### To build from source
```
tar -xzf mattermost-torq-sync.tar.gz
cd mattermost-torq-sync
go mod tidy
make dist
```

### Deploy 
Upload dist/com.yourorg.torq-sync.tar.gz via System Console → Plugins → Plugin Management, then configure the Torq webhook URL, secret key, and secret value in the plugin's settings page.
On the Torq side: create an HTTP webhook trigger, add a condition checking that the secret key and value matches your configured value, then branch on event_type in the payload.


## VERSION HISTORY
0.5.0 - 2026-06-25 - Bugfixes
0.2.0 - 2026-06-22 - Clean-up, additional improvements
0.1.0 - 2026-06-18 - Initial draft

## DEVELOPMENT NOTES
Parts of this PlugIn were written with the assistance of generative AI.