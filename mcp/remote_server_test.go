// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// rawMCPServer is a hand-written streamable-HTTP MCP endpoint. The SDK's own
// server always answers with a version the SDK itself supports, so it cannot
// express a peer that negotiates a different one. Hanzo cloud's fleet door
// (POST /v1/mcp) does exactly that, which is what these tests reproduce.
type rawMCPServer struct {
	protocolVersion string
	// toolsListResult is the verbatim JSON for the tools/list result.
	toolsListResult string
}

func (s rawMCPServer) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// The client opens a GET SSE stream after initializing; it treats a
		// refusal as "this server has no server-initiated stream".
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Notifications carry no id and expect no body.
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var result string
	switch req.Method {
	case "initialize":
		result = fmt.Sprintf(
			`{"capabilities":{"tools":{"listChanged":false}},"protocolVersion":%q,"serverInfo":{"name":"raw","version":"1.0.0"}}`,
			s.protocolVersion,
		)
	case "tools/list":
		result = s.toolsListResult
	default:
		result = `{}`
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
}

func (s rawMCPServer) start(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(server.Close)
	return server
}

const oneToolResult = `{"tools":[{"name":"tool_1","description":"Test tool tool_1","inputSchema":{"type":"object"}}]}`

// TestClientConnectsToNegotiatedProtocolVersions pins the set of MCP protocol
// versions a peer may negotiate and still be usable. Hanzo cloud's fleet door
// answers initialize with 2026-07-28 regardless of what the client asks for, so
// an SDK that predates that version rejects the handshake and every tool behind
// the door — code execution, sandboxes, research — becomes unreachable.
func TestClientConnectsToNegotiatedProtocolVersions(t *testing.T) {
	for _, tc := range []string{
		"2026-07-28", // Hanzo cloud fleet door
		"2025-11-25",
		"2025-06-18",
		"2025-03-26",
		"2024-11-05",
	} {
		t.Run(tc, func(t *testing.T) {
			httpServer := rawMCPServer{protocolVersion: tc, toolsListResult: oneToolResult}.start(t)

			client, err := NewClient(context.Background(), "user-id", ServerConfig{
				Name:    "raw",
				BaseURL: httpServer.URL,
				Enabled: true,
			}, newTestLogService(), newTestOAuthManager(), httpServer.Client(), newTestToolsCache(), false)
			require.NoError(t, err, "protocol version %s must be usable", tc)
			t.Cleanup(func() { _ = client.Close() })

			require.Len(t, client.Tools(), 1)
		})
	}
}

// TestClientUsesDefaultTransportWhenUnset covers an http.Client whose Transport
// is nil, which net/http reads as http.DefaultTransport. The header and OAuth
// wrappers call the base themselves, so a nil one has to be resolved before it
// reaches them.
func TestClientUsesDefaultTransportWhenUnset(t *testing.T) {
	httpServer := rawMCPServer{protocolVersion: "2026-07-28", toolsListResult: oneToolResult}.start(t)

	client, err := NewClient(context.Background(), "user-id", ServerConfig{
		Name:    "raw",
		BaseURL: httpServer.URL,
		Enabled: true,
		Headers: map[string]string{"Authorization": "Bearer token"},
	}, newTestLogService(), newTestOAuthManager(), &http.Client{}, newTestToolsCache(), false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	require.Len(t, client.Tools(), 1)
}

// TestClientSendsConfiguredHeaders pins that a static bearer set by an admin on
// the server config reaches the server on every request, which is how the plugin
// authenticates to an MCP endpoint that does not run the OAuth dance.
func TestClientSendsConfiguredHeaders(t *testing.T) {
	seen := make(chan string, 8)
	server := rawMCPServer{protocolVersion: "2026-07-28", toolsListResult: oneToolResult}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case seen <- r.Header.Get("Authorization"):
		default:
		}
		server.handle(w, r)
	}))
	t.Cleanup(httpServer.Close)

	client, err := NewClient(context.Background(), "user-id", ServerConfig{
		Name:    "raw",
		BaseURL: httpServer.URL,
		Enabled: true,
		Headers: map[string]string{"Authorization": "Bearer sekret"},
	}, newTestLogService(), newTestOAuthManager(), httpServer.Client(), newTestToolsCache(), false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.Len(t, client.Tools(), 1)

	close(seen)
	require.NotEmpty(t, seen, "server received no requests")
	for got := range seen {
		require.Equal(t, "Bearer sekret", got)
	}
}

// TestClientSurvivesNullToolInListing covers a server that puts a JSON null in
// its tools array. listAllTools skips nil tools, but that guard is only
// reachable if nothing dereferences the entry first.
func TestClientSurvivesNullToolInListing(t *testing.T) {
	httpServer := rawMCPServer{
		protocolVersion: "2026-07-28",
		toolsListResult: `{"tools":[null,{"name":"tool_1","description":"Test tool tool_1","inputSchema":{"type":"object"}}]}`,
	}.start(t)

	client, err := NewClient(context.Background(), "user-id", ServerConfig{
		Name:    "raw",
		BaseURL: httpServer.URL,
		Enabled: true,
	}, newTestLogService(), newTestOAuthManager(), httpServer.Client(), newTestToolsCache(), false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	tools := client.Tools()
	require.Len(t, tools, 1)
	require.Contains(t, tools, "tool_1")
}
