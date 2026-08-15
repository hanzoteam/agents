# The Agents plugin, as an artifact image.
#
# This image runs nothing. It carries one file — the built plugin bundle — so
# that something else can install it. app.hanzo.team runs it as an init
# container that copies the bundle into the workspace's file store, where the
# server's own plugin sync finds and installs it on boot.
#
# A file store install rather than a prepackaged one: buildPrepackagedPlugin
# rejects a prepackaged bundle that carries no signature, and verifies the
# signature against Mattermost's key, which a fork cannot produce. syncPlugins
# reads the file store and asks for a signature only when RequirePluginSignature
# is set. So the bundle goes where a plugin an operator installed goes, because
# that is what this is.

FROM golang:1.26.4-bookworm AS manifest
WORKDIR /src
COPY . .
# server/manifest.go and webapp/src/manifest.ts are generated from plugin.json
# and gitignored, so neither half compiles until this has run.
RUN go run ./build/manifest apply

# Both architectures, because plugin.json names both and the server picks by the
# platform it runs on — a bundle carrying one is simply absent on the other.
FROM golang:1.26.4-bookworm AS server
COPY --from=manifest /src /src
WORKDIR /src/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/plugin-linux-amd64 \
 && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o dist/plugin-linux-arm64

FROM node:24.11-bookworm AS webapp
COPY --from=manifest /src /src
WORKDIR /src/webapp
RUN npm ci --no-audit --no-fund && npm run build

# The layout is the one `make bundle` produces: everything under a single
# directory named for the plugin id, which is what extractPlugin expects to
# find at the root of the archive.
FROM busybox:1.37 AS bundle
WORKDIR /bundle/mattermost-ai
COPY --from=manifest /src/plugin.json /src/LICENSE.txt /src/NOTICE.txt ./
COPY --from=manifest /src/assets ./assets
COPY --from=manifest /src/public ./public
COPY --from=server /src/server/dist ./server/dist
COPY --from=webapp /src/webapp/dist ./webapp/dist
RUN cd /bundle && tar -czf /mattermost-ai.tar.gz mattermost-ai

# busybox rather than scratch: the init container needs `cp`, and the whole job
# is to copy this one file somewhere the server reads.
FROM busybox:1.37
COPY --from=bundle /mattermost-ai.tar.gz /bundle/mattermost-ai.tar.gz
