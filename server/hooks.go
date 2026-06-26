// Torq Private Channel Sync
//
// hooks.go
// The core logic: filters every event to private channels only, then forwards defined event types
//
// version 1.1.1

package main

import (
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// isPrivateChannel reports whether the given channel ID refers to a private
// channel ("P" type in Mattermost's model). Group messages ("G") and direct
// messages ("D") are deliberately excluded here -- treat those as a separate
// decision since they have different membership/consent semantics. Adjust
// this if your Torq use case should also cover DMs/GMs.

func (p *Plugin) isPrivateChannel(channelID string) (*model.Channel, bool) {
	channel, appErr := p.API.GetChannel(channelID)
	if appErr != nil {
		p.API.LogWarn("Torq Sync: failed to look up channel", "channel_id", channelID, "err", appErr.Error())
		return nil, false
	}
	return channel, channel.Type == model.ChannelTypePrivate
}

func (p *Plugin) shouldForward(channelID string) (*model.Channel, bool) {
	cfg := p.getConfiguration()
	if cfg.TorqWebhookURL == "" {
		return nil, false
	}

	channel, isPrivate := p.isPrivateChannel(channelID)
	if !isPrivate {
		return nil, false
	}

	if !cfg.channelAllowed(channelID) {
		return nil, false
	}

	return channel, true
}

// MessageHasBeenPosted fires after a post is committed to the database.
// Minimum server version: 5.2

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if post.IsSystemMessage() {
		return
	}

	channel, ok := p.shouldForward(post.ChannelId)
	if !ok {
		return
	}

	cfg := p.getConfiguration()
	evt := eventEnvelope{
		EventType:          "post_created",
		Timestamp:          time.Now().UnixMilli(),
		TeamID:             channel.TeamId,
		ChannelID:          post.ChannelId,
		ChannelDisplayName: channel.DisplayName,
		UserID:             post.UserId,
		PostID:             post.Id,
		PostType:           post.Type,
		FileIDs:            post.FileIds,
		HasAttachments:     len(post.FileIds) > 0,
		RootID:             post.RootId,
		IsReply:            post.RootId != "",
		ReplyCount:         post.ReplyCount,
	}
	if cfg.IncludeMessageContent {
		evt.Message = post.Message
	}
	p.torqClient.send(evt)
}

// UserHasJoinedChannel fires after a user is added to a channel, whether by
// themselves, an admin, or a plugin/bot.
// Minimum server version: 5.2

func (p *Plugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	channel, ok := p.shouldForward(channelMember.ChannelId)
	if !ok {
		return
	}

	evt := eventEnvelope{
		EventType:    "channel_member_joined",
		Timestamp:    time.Now().UnixMilli(),
		TeamID:       channel.TeamId,
		ChannelID:    channelMember.ChannelId,
		ChannelDisplayName: channel.DisplayName,
		TargetUserID: channelMember.UserId,
	}
	if actor != nil {
		evt.UserID = actor.Id
	}

	p.torqClient.send(evt)
}

// UserHasLeftChannel fires after a user is removed from or leaves a channel.
// Minimum server version: 5.2

func (p *Plugin) UserHasLeftChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	channel, ok := p.shouldForward(channelMember.ChannelId)
	if !ok {
		return
	}

	evt := eventEnvelope{
		EventType:    "channel_member_left",
		Timestamp:    time.Now().UnixMilli(),
		TeamID:       channel.TeamId,
		ChannelID:    channelMember.ChannelId,
		ChannelDisplayName: channel.DisplayName,
		TargetUserID: channelMember.UserId,
	}
	if actor != nil {
		evt.UserID = actor.Id
	}

	p.torqClient.send(evt)
}

// MessageHasBeenUpdated fires after an edited post is committed to the database.
// Minimum server version: 5.2

func (p *Plugin) MessageHasBeenUpdated(c *plugin.Context, newPost, oldPost *model.Post) {
	if newPost.IsSystemMessage() {
		return
	}

	channel, ok := p.shouldForward(newPost.ChannelId)
	if !ok {
		return
	}

	cfg := p.getConfiguration()
	evt := eventEnvelope{
		EventType:          "post_updated",
		Timestamp:          time.Now().UnixMilli(),
		TeamID:             channel.TeamId,
		ChannelID:          newPost.ChannelId,
		ChannelDisplayName: channel.DisplayName,
		UserID:             newPost.UserId,
		PostID:             newPost.Id,
		PostType:           newPost.Type,
		FileIDs:            newPost.FileIds,
		HasAttachments:     len(newPost.FileIds) > 0,
		EditedAt:           newPost.EditAt,
		RootID:             newPost.RootId,
		IsReply:            newPost.RootId != "",
		ReplyCount:         newPost.ReplyCount,
	}
	if cfg.IncludeMessageContent {
		evt.Message = newPost.Message
		evt.Extra = map[string]any{
			"previous_message": oldPost.Message,
		}
	}

	p.torqClient.send(evt)
}

// MessageHasBeenDeleted fires after a post is marked as deleted in the database
// (Mattermost soft-deletes posts; the row persists with DeleteAt set). Note this
// fires for posts deleted by plugins too, including this one if it ever deletes
// posts itself.
// Minimum server version: 5.26 -- confirm against your server version before relying on this.

func (p *Plugin) MessageHasBeenDeleted(c *plugin.Context, post *model.Post) {
	if post.IsSystemMessage() {
		return
	}
 
	channel, ok := p.shouldForward(post.ChannelId)
	if !ok {
		return
	}
 
	cfg := p.getConfiguration()
	evt := eventEnvelope{
		EventType:          "post_deleted",
		Timestamp:          time.Now().UnixMilli(),
		TeamID:             channel.TeamId,
		ChannelID:          post.ChannelId,
		ChannelDisplayName: channel.DisplayName,
		UserID:             post.UserId,
		PostID:             post.Id,
		PostType:           post.Type,
		FileIDs:            post.FileIds,
		HasAttachments:     len(post.FileIds) > 0,
		EditedAt:           post.DeleteAt,
		RootID:             post.RootId,
		IsReply:            post.RootId != "",
		ReplyCount:         post.ReplyCount,
	}
	if cfg.IncludeMessageContent {
		evt.Message = post.Message
	}
 
	p.torqClient.send(evt)
}

// ReactionHasBeenAdded fires after a reaction is committed to the database.
// Note: this fires for reactions added by plugins too, including this one.
// Minimum server version: 5.30

func (p *Plugin) ReactionHasBeenAdded(c *plugin.Context, reaction *model.Reaction) {
	p.forwardReactionEvent("reaction_added", reaction)
}

// ReactionHasBeenRemoved fires after a reaction's removal is committed to the database.
// Minimum server version: 5.30

func (p *Plugin) ReactionHasBeenRemoved(c *plugin.Context, reaction *model.Reaction) {
	p.forwardReactionEvent("reaction_removed", reaction)
}

// forwardReactionEvent resolves the reaction's post to find its channel (Reaction
// itself doesn't carry ChannelId), applies the same private-channel filter as
// other events, and dispatches to Torq.

func (p *Plugin) forwardReactionEvent(eventType string, reaction *model.Reaction) {
	post, appErr := p.API.GetPost(reaction.PostId)
	if appErr != nil {
		p.API.LogWarn("Torq Sync: failed to look up post for reaction", "post_id", reaction.PostId, "err", appErr.Error())
		return
	}

	channel, ok := p.shouldForward(post.ChannelId)
	if !ok {
		return
	}

	evt := eventEnvelope{
		EventType:          eventType,
		Timestamp:          time.Now().UnixMilli(),
		TeamID:             channel.TeamId,
		ChannelID:          post.ChannelId,
		ChannelDisplayName: channel.DisplayName,
		UserID:             reaction.UserId,
		PostID:             reaction.PostId,
		PostType:           post.Type,
		FileIDs:            post.FileIds,
		HasAttachments:     len(post.FileIds) > 0,
		RootID:             post.RootId,
		IsReply:            post.RootId != "",
		ReplyCount:         post.ReplyCount,
		Extra: map[string]any{
			"emoji_name": reaction.EmojiName,
		},
	}

	p.torqClient.send(evt)
}

// ChannelHasBeenCreated fires after a new channel is created.
// Minimum server version: 5.2

func (p *Plugin) ChannelHasBeenCreated(c *plugin.Context, channel *model.Channel) {
	if channel.Type != model.ChannelTypePrivate {
		return
	}
	cfg := p.getConfiguration()
	if cfg.TorqWebhookURL == "" || !cfg.channelAllowed(channel.Id) {
		return
	}

	evt := eventEnvelope{
		EventType: "channel_created",
		Timestamp: time.Now().UnixMilli(),
		TeamID:    channel.TeamId,
		ChannelID: channel.Id,
		ChannelDisplayName: channel.DisplayName,
		UserID:    channel.CreatorId,
	}

	p.torqClient.send(evt)
}
