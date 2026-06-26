PLUGIN_ID := mattermost-torq-private-sync
BUNDLE_NAME := $(PLUGIN_ID).tar.gz

GO := go
GOFLAGS := CGO_ENABLED=0

.PHONY: all dist server clean

all: dist

server:
	mkdir -p server/dist
	$(GOFLAGS) GOOS=linux  GOARCH=amd64 $(GO) build -o server/dist/plugin-linux-amd64   ./server
	$(GOFLAGS) GOOS=linux  GOARCH=arm64 $(GO) build -o server/dist/plugin-linux-arm64   ./server
	$(GOFLAGS) GOOS=darwin GOARCH=amd64 $(GO) build -o server/dist/plugin-darwin-amd64  ./server
	$(GOFLAGS) GOOS=darwin GOARCH=arm64 $(GO) build -o server/dist/plugin-darwin-arm64  ./server

dist: server
	mkdir -p dist/$(PLUGIN_ID)/server/dist
	cp plugin.json dist/$(PLUGIN_ID)/
	cp server/dist/plugin-* dist/$(PLUGIN_ID)/server/dist/
	cd dist && tar -czf $(BUNDLE_NAME) $(PLUGIN_ID)
	@echo "Bundle ready at dist/$(BUNDLE_NAME) -- upload via System Console > Plugins > Plugin Management"

clean:
	rm -rf server/dist dist
