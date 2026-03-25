package internal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldShowBlockingPrompt(t *testing.T) {
	t.Run("never dismissed", func(t *testing.T) {
		info := UpdateInfo{
			AvailableVersion: "1.2.0",
		}
		assert.True(t, shouldShowBlockingPrompt(info))
	})

	t.Run("dismissed different version", func(t *testing.T) {
		info := UpdateInfo{
			AvailableVersion: "1.3.0",
			DismissedVersion: "1.2.0",
			DismissedAt:      time.Now(),
		}
		assert.True(t, shouldShowBlockingPrompt(info))
	})

	t.Run("dismissed same version recently", func(t *testing.T) {
		info := UpdateInfo{
			AvailableVersion: "1.2.0",
			DismissedVersion: "1.2.0",
			DismissedAt:      time.Now().Add(-10 * time.Minute),
		}
		assert.False(t, shouldShowBlockingPrompt(info))
	})

	t.Run("dismissed same version expired", func(t *testing.T) {
		info := UpdateInfo{
			AvailableVersion: "1.2.0",
			DismissedVersion: "1.2.0",
			DismissedAt:      time.Now().Add(-31 * time.Minute),
		}
		assert.True(t, shouldShowBlockingPrompt(info))
	})
}

func TestUpdateCommandHint(t *testing.T) {
	tests := []struct {
		method   InstallMethod
		expected string
	}{
		{InstallMethodHomebrew, "brew upgrade tdl"},
		{InstallMethodGoInstall, "go install github.com/ThreeDotsLabs/cli/tdl@latest"},
		{InstallMethodNix, "nix profile upgrade --flake github:ThreeDotsLabs/cli"},
		{InstallMethodScoop, "scoop update tdl"},
		{InstallMethodDirectBinary, ""},
		{InstallMethodUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, updateCommandHint(tt.method))
		})
	}
}

func TestUpdateInfoBackwardCompatibility(t *testing.T) {
	t.Run("old JSON without new fields deserializes cleanly", func(t *testing.T) {
		oldJSON := `{"current_version":"1.0.0","available_version":"1.1.0","update_available":true,"last_checked":"2025-01-01T00:00:00Z"}`
		var info UpdateInfo
		err := json.Unmarshal([]byte(oldJSON), &info)
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", info.CurrentVersion)
		assert.Equal(t, "1.1.0", info.AvailableVersion)
		assert.True(t, info.UpdateAvailable)
		// New fields default to zero values
		assert.Empty(t, info.ReleaseNotes)
		assert.Empty(t, info.DismissedVersion)
		assert.True(t, info.DismissedAt.IsZero())
	})

	t.Run("round trip with all fields", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		info := UpdateInfo{
			CurrentVersion:   "1.0.0",
			AvailableVersion: "1.1.0",
			UpdateAvailable:  true,
			LastChecked:      now,
			ReleaseNotes:     "- bug fix",
			DismissedVersion: "1.1.0",
			DismissedAt:      now,
		}
		data, err := json.Marshal(info)
		require.NoError(t, err)

		var decoded UpdateInfo
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, info.CurrentVersion, decoded.CurrentVersion)
		assert.Equal(t, info.AvailableVersion, decoded.AvailableVersion)
		assert.Equal(t, info.ReleaseNotes, decoded.ReleaseNotes)
		assert.Equal(t, info.DismissedVersion, decoded.DismissedVersion)
		assert.True(t, info.DismissedAt.Equal(decoded.DismissedAt))
	})
}
