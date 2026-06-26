# Torq Mattermost Private Channel Webhook PlugIn

## PURPOSE
Mattermost's classic Outgoing Webhooks feature is scoped to a specific channel (or all channels in a team for public ones), but it's designed around trigger-word matching on messages in public channels only. There's no admin console option to attach an outgoing webhook to a private channel; this is a deliberate product limitation, not a config you're missing.
So for private channel data, you're left with two real options: the websocket, or Mattermost's Plugin system, which can run server-side and isn't bound by that public-channel restriction.

Since Torq doesn't hold a persistent inbound connection (Torq does not support websockets), we require something on the Mattermost side (or adjacent to it) that does,  which then talks to Torq over native HTTP request/response — which Torq handles natively via its webhook trigger.

Mattermost supports server plugins that hook directly into the server's internal event system (not the websocket, the actual Go-level Hooks interface (`MessageHasBeenPosted`, `ChannelHasBeenCreated`, `UserHasJoinedChannel`, etc.)). This runs inside the Mattermost server process, sees private channel events natively (no membership/visibility issue), and can make an outbound HTTP/S POST to a Torq webhook URL per event.

There is no persistent connection problem at all, as it's an in-process hook, not a socket.

This PlugIn aligns Mattermost private channels with the native Webhook notification capabilities with public channels.

This requires System Admin access to install, and someone to maintain a small Go plugin (this feature).

#### The list of activities this PlugIn handles
* `MessageHasBeenPosted`
* `UserHasJoinedChannel`
* `UserHasLeftChannel`
* `MessageHasBeenUpdated`
* `MessageHasBeenDeleted`
   * Note: Mattermost soft-deletes posts; the row persists with DeleteAt set). This fires for posts deleted by plugins, too, including this one, if it ever deletes posts itself.
* `ReactionHasBeenAdded`
* `ReactionHasBeenRemoved`
   * Important note about ReactionEvents: Reaction events don't carry ChannelId. For ReactionEvents, the PlugIn performs an extra `p.API.GetPost` call per reaction event to resolve the channel for inclusion in the webhook; this adds a modest amount of extra API/database load per reaction.  

##
### System Requirements
This PlugIn requires a Mattermost minimum server version of 7.8.0

##
## Building and deploying

### To build from source
```
tar -xzf mattermost-torq-private-sync.tar.gz // git clone from the repository
cd mattermost-torq-sync
go mod tidy
make dist
```
This will produce the `mattermost-torq-private-sync.tar.gz` required for deployment.

### Deploy 
#### In Torq
Create an HTTP webhook trigger, add your desired Authentication shared secret header (optional).  

#### In Mattermost
##### TL;DR: 
Upload `mattermost-torq-private-sync.tar.gz` via System Console → Plugins → Plugin Management 

##### Step-by-step:
1. Navigate to the System Console
2. Scroll to the PlugIns Section on the left Nav Bar
3. Click on `PlugIn Management` in that section
4. On the `PlugIn Management` configuration screen
   1. Ensure that `Enable Plugins` is set to `True`
   2. Navigate to the `Upload Plugin` section and click the `Choose File` button
   3. Select the mattermost-torq-private-sync.tar.gz file from the file picker
   4. Click the "Upload" button
5. If completed successfully, you should see "Successfully uploaded plugin from mattermost-torq-private-sync.tar.gz" 

### Configure
Once deployed, `Torq Private Channel Sync` should appear in the Nav Sidebar under the `PlugIns` section. Navigate to this section to configure the PlugIn

##### Step-by-step:
1. Navigate to the System Console
2. Scroll to the PlugIns Section on the left Nav Bar
3. Click on "Torq Private Channel Sync" in that section
4. Set `Enable Plugin` to `True`
5. Paste your Torq Webhook URL into the `Torq Webhook URL` dialogue box (generally `https://hooks.[eu.|jp.]torq.io/v1/webhooks/<UUID>`)
6. (Optional but recommended) Add an Authentication Header (shared secret) as configured in your Torq Webhook
   * One important note:  the secret will display on the screen until the window is refreshed or you navigate away from and back to the PlugIn configuration page.
7. Select whether to sync all private channels or selected channels only
   * In `Sync all private channels` set either `True` or `False`
   * If `False`, choose the channels to sync in the `Channel ID allow-list (comma-separated)` parameter
8) Choose whether to include the message body in the trigger payload.  
   * In `Include message body in payload` set either `True` or `False`
   * If `True`, the message will be included in the trigger payload. If `False` only metadata (channel, user, post ID, timestamp) is sent.  A separate action will be required to pull the payload. 
9) Click `Save` to save your settings
10) Navigate back to the `PlugIn Management` configuration screen.  Scroll to the bottom of the page, to the `Installed Plugins` section.  You should see `Torq Private Channel Sync (mattermost-torq-private-sync - <version>)` listed and a green checked `This plugin is running` notice.

##   
## Using
Once configured, messages posted in private channels will be transmitted to the configured webhook. 

In your workflow(s) using using the webhook trigger,  branch on the payload data for your desired activities (such as `event_type` or `channel_id`).

Note: Mattermost won't surface private channel content to a bot or webhook that hasn't been added, even with admin-level API tokens in some configurations. If you wish for the bot to interact with a private channel, you'll need a provisioning step (manual or scripted via POST /api/v4/channels/{channel_id}/members) to add your bot to each private channel in which it should interact

## 
### Sample data
#### Post Created
```
{
  "timestamp_ms": 1782511625322,
  "channel_display_name": "My Private Channel",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "event_type": "post_created",
  "is_reply": false,
  "message": "Hello world!",
  "has_attachments": false,
  "post_id": "uorzhb4ni3rc9edkm7sa8w8wkh"
}
```

#### Post with File Attachment Created
```
{
  "timestamp_ms": 1782512607912,
  "channel_display_name": "My Private Channel",
  "message": "Ask, and ye shall receive @torqbot ",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "is_reply": false,
  "has_attachments": true,
  "post_id": "9fot1ag55ffhdqnid1maytugka",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "file_ids": [
    "akyzkpkgi7g9fdmjkkui8tcwbe"
  ],
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "event_type": "post_created"
}
```

#### Response Post Created
```
{
  "post_id": "zkgw55txiprgixd6sifhk1mjzr",
  "has_attachments": false,
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "reply_count": 1,
  "root_post_id": "uorzhb4ni3rc9edkm7sa8w8wkh",
  "message": "Hello to you! 👋 ",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "is_reply": true,
  "event_type": "post_created",
  "channel_display_name": "My Private Channel",
  "timestamp_ms": 1782511832660
}
```

#### Post Updated
```
{
  "timestamp_ms": 1782511765305,
  "has_attachments": false,
  "message": "שלום ",
  "post_id": "uorzhb4ni3rc9edkm7sa8w8wkh",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "channel_display_name": "My Private Channel",
  "extra": {
    "previous_message": "Greetings, Earthlings!"
  },
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "is_reply": false,
  "edited_at": 1782511765300,
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "event_type": "post_updated"
}
```

#### Response Post Updated
```
{
  "channel_display_name": "My Private Channel",
  "message": "Thank you 👍 !",
  "post_id": "zkgw55txiprgixd6sifhk1mjzr",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "is_reply": true,
  "event_type": "post_updated",
  "has_attachments": false,
  "extra": {
    "previous_message": "Hello to you! 👋 "
  },
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "edited_at": 1782511927137,
  "root_post_id": "uorzhb4ni3rc9edkm7sa8w8wkh",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "reply_count": 1,
  "timestamp_ms": 1782511927146
}
```

#### Post Deleted
```
{
  "channel_display_name": "My Private Channel",
  "message": "Hi there",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "event_type": "post_deleted",
  "timestamp_ms": 1782513480503,
  "has_attachments": false,
  "post_id": "rk4tikrkrjyr8birwiekgqxxky",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "is_reply": false,
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o"
}
```
#### Response Post Deleted
```
{
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "post_id": "i1fgtzc7k3fi5dzad86yqr9rte",
  "is_reply": true,
  "timestamp_ms": 1782513541686,
  "message": "Cheers!",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "reply_count": 2,
  "channel_display_name": "My Private Channel",
  "event_type": "post_deleted",
  "root_post_id": "uorzhb4ni3rc9edkm7sa8w8wkh",
  "has_attachments": false
}
```

#### Reaction Added
```
{
  "is_reply": false,
  "timestamp_ms": 1782511685572,
  "channel_display_name": "My Private Channel",
  "extra": {
    "emoji_name": "wave"
  },
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "has_attachments": false,
  "post_id": "uorzhb4ni3rc9edkm7sa8w8wkh",
  "event_type": "reaction_added",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar"
}
```

#### Reaction Removed
```
{
  "extra": {
    "emoji_name": "wave"
  },
  "channel_id": "5fqak1si8irgz8f845gfhbzh8o",
  "event_type": "reaction_removed",
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "timestamp_ms": 1782511724134,
  "channel_display_name": "My Private Channel",
  "post_id": "uorzhb4ni3rc9edkm7sa8w8wkh",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "has_attachments": false,
  "is_reply": false
}
```

#### Channel Created
```
{
  "event_type": "channel_created",
  "timestamp_ms": 1782486052424,
  "extra": {
    "display_name": "Another Private Channel"
  },
  "team_id": "38zj3sdt5fbxmniu7q8qmafrsh",
  "user_id": "3tr7wn3nzj8qzqz58pek5dmuar",
  "channel_id": "urwxh1pi4tbhfmuhx33q4pnpxy"
}
```

##
## Disabling or Removing (uninstalling) the PlugIn
In the unlikely event that you choose to remove the PlugIn, the process is simple:

1. Navigate to the System Console
2. Scroll to the PlugIns Section on the left Nav Bar
3. Click on `PlugIn Management` in that section
4. On the `PlugIn Management` configuration screen, scroll to the bottom of the menu, to the `Installed Plugins` section
   * You should see `Torq Private Channel Sync (mattermost-torq-private-sync - <version>)` listed and a green checked `This plugin is running` notice.
5. Select the action you wish to take on the PlugIn - `Disable` or `Remove`
   * You may also disable the PlugIn from the `Torq Private Channel Sync` PlugIn Settings page, by setting `Enable Plugin` to `False`

##
## VERSION HISTORY
* 1.2.0 - 2026-06-26 - Add handler for `MessageHasBeenDeleted` events
* 1.1.0 - 2026-06-26 - Expand eventEnvelope payload to include additional fields
* 1.0.0 - 2026-06-26 - Initial release
* 0.7.0 - 2026-06-26 - Extended to include message updates and reactions
* 0.6.0 - 2026-06-26 - Bugfixes and copy
* 0.5.0 - 2026-06-25 - Bugfixes
* 0.2.0 - 2026-06-22 - Clean-up, additional improvements
* 0.1.0 - 2026-06-18 - Initial draft

##
## DEVELOPMENT NOTES
Parts of this PlugIn were written with the assistance of generative AI.
##