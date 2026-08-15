// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package config

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-agents/v2/llm"
	"github.com/stretchr/testify/require"
)

func TestServiceKeyEnv(t *testing.T) {
	// A service id is free-form; a variable name is not.
	for id, want := range map[string]string{
		"hanzo":       "AGENTS_KEY_HANZO",
		"openai":      "AGENTS_KEY_OPENAI",
		"my-service":  "AGENTS_KEY_MY_SERVICE",
		"vertex.ai 2": "AGENTS_KEY_VERTEX_AI_2",
		"":            "AGENTS_KEY_",
	} {
		require.Equal(t, want, serviceKeyEnv(id), "id %q", id)
	}
}

func TestSuppliesKeysFromEnv(t *testing.T) {
	t.Run("an empty key is taken from the environment", func(t *testing.T) {
		t.Setenv("AGENTS_KEY_HANZO", "from-kms")

		var c Container
		c.Update(&Config{Services: []llm.ServiceConfig{{ID: "hanzo"}}})

		svc, ok := c.GetServiceByID("hanzo")
		require.True(t, ok)
		require.Equal(t, "from-kms", svc.APIKey)
	})

	t.Run("a configured key is left alone", func(t *testing.T) {
		t.Setenv("AGENTS_KEY_HANZO", "from-kms")

		var c Container
		c.Update(&Config{Services: []llm.ServiceConfig{{ID: "hanzo", APIKey: "configured"}}})

		svc, _ := c.GetServiceByID("hanzo")
		require.Equal(t, "configured", svc.APIKey, "the environment must not override an admin's own value")
	})

	t.Run("no variable leaves the key empty", func(t *testing.T) {
		var c Container
		c.Update(&Config{Services: []llm.ServiceConfig{{ID: "absent"}}})

		svc, _ := c.GetServiceByID("absent")
		require.Empty(t, svc.APIKey)
	})

	t.Run("the key does not reach what is stored", func(t *testing.T) {
		// The whole point of resolving into the copy: an admin saving the
		// settings page must not write the credential into the config table.
		t.Setenv("AGENTS_KEY_HANZO", "from-kms")

		stored := &Config{Services: []llm.ServiceConfig{{ID: "hanzo"}}}
		var c Container
		c.Update(stored)

		require.Empty(t, stored.Services[0].APIKey, "the caller's configuration was mutated")
	})

	t.Run("the path that reloads from storage resolves too", func(t *testing.T) {
		t.Setenv("AGENTS_KEY_HANZO", "from-kms")

		var c Container
		require.NoError(t, c.StorePersistedConfigWithoutNotify(
			&Config{Services: []llm.ServiceConfig{{ID: "hanzo"}}}))

		svc, _ := c.GetServiceByID("hanzo")
		require.Equal(t, "from-kms", svc.APIKey)
	})
}
