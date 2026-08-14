// Package configinit writes a default .nopiirc.toml for nopii init config.
package configinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/iilei/nopii/internal/config"
)

const (
	// fileMode is the permission bits for the generated config file.
	fileMode = 0o600

	// Template is the default config written by `nopii init config`.
	// It contains every supported key with its default value and brief comments.
	Template = `version = 1

# scope ties pseudonyms together: same scope + key => same token across runs.
# Commit this file to share consistent pseudonymization policy across a team.
scope = "default"

[key]
env = "NOPII_KEY"
# file = "/run/secrets/nopii-key"

[output]
token_length = 12

[recognizers]
email = true
ipv4  = true
uuid  = true
phone = true

[git.date_clamp]
enabled             = false
granularity_seconds = 86400  # floor timestamps to this boundary (e.g. 86400 = daily, 7776000 = 90 days)
`
)

// Create writes a default .nopiirc.toml in dir.
// Returns an error if the file already exists and force is false.
func Create(dir string, force bool) (string, error) {
	path := filepath.Join(dir, config.FileName)
	if _, err := os.Stat(path); err == nil && !force {
		return "", fmt.Errorf("%s already exists; use --force to overwrite", path)
	}
	if err := os.WriteFile(path, []byte(Template), fileMode); err != nil {
		return "", err
	}
	return path, nil
}
