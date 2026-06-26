// Torq Private Channel Sync
//
// main.go
// lifecycle plumbing
//
// version 1.1.0

package main

import (
	"github.com/mattermost/mattermost/server/public/plugin"
)

func main() {
	plugin.ClientMain(&Plugin{})
}
