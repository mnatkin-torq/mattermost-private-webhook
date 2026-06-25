// Torq Private Channel Sync
// version 0.5.0

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
		EventType: "post_created",
		Timestamp: time.Now().UnixMilli(),
		TeamID:    channel.TeamId,
		ChannelID: post.ChannelId,
		UserID:    post.UserId,
		PostID:    post.Id,
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
		TargetUserID: channelMember.UserId,
	}
	if actor != nil {
		evt.UserID = actor.Id
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
		UserID:    channel.CreatorId,
		Extra: map[string]any{
			"display_name": channel.DisplayName,
		},
	}

	p.torqClient.send(evt)
}
