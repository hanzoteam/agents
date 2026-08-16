// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package assets

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

// wantAgents is the canonical list of built-in agents that must resolve to a
// role-matched icon. It is independent of agentProfilePictures so that an
// accidental deletion from the map fails this test instead of silently
// shrinking what gets checked.
var wantAgents = []string{
	"vi", "dev", "des", "opera", "db", "sec", "core", "algo",
	"mark", "su", "fin", "cal", "art", "mu", "data", "chat",
}

func TestAgentProfilePictureDecodesForEveryAgent(t *testing.T) {
	for _, name := range wantAgents {
		t.Run(name, func(t *testing.T) {
			data := AgentProfilePicture(name)
			require.NotEmpty(t, data)
			require.NotEqual(t, DefaultAgentProfilePicture, data,
				"agent %q must have its own icon, not the fallback", name)

			img, err := png.Decode(bytes.NewReader(data))
			require.NoError(t, err, "must decode as PNG")
			require.Equal(t, 128, img.Bounds().Dx(), "width must be 128px")
			require.Equal(t, 128, img.Bounds().Dy(), "height must be 128px")
		})
	}
}

func TestAgentProfilePictureUnknownAgentFallsBackToDefault(t *testing.T) {
	require.Equal(t, DefaultAgentProfilePicture, AgentProfilePicture("not-a-real-agent"))
	require.Equal(t, DefaultAgentProfilePicture, AgentProfilePicture(""))
}

func TestAgentProfilePicturesAreAllDistinct(t *testing.T) {
	icons := make(map[string][]byte, len(wantAgents))
	for _, name := range wantAgents {
		icons[name] = AgentProfilePicture(name)
	}
	for i, a := range wantAgents {
		for _, b := range wantAgents[i+1:] {
			require.False(t, bytes.Equal(icons[a], icons[b]),
				"agent %q and %q must not share the same icon", a, b)
		}
	}
}
