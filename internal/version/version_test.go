package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionMatchesReleaseManifests(t *testing.T) {
	for _, path := range []string{".claude-plugin/plugin.json", ".codex-plugin/plugin.json", "package.json", "plugin/plugin.json", "plugin/.claude-plugin/plugin.json", "plugin/.codex-plugin/plugin.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Version != Version {
			t.Fatalf("%s: %s, running version %s", path, manifest.Version, Version)
		}
	}
}
