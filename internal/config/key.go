package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

// ResolveKey returns the pseudonymization key bytes and a human-readable
// source description.
func ResolveKey(cfg *Config, envOverride, fileOverride string, stdinKey []byte) ([]byte, string, error) {
	if cfg == nil {
		return nil, "", errors.New("nil config")
	}
	if len(stdinKey) > 0 {
		k := bytes.TrimSpace(stdinKey)
		if len(k) == 0 {
			return nil, "", errors.New("empty key from stdin")
		}
		return k, "stdin", nil
	}
	if fileOverride != "" {
		// #nosec G304 -- file path comes from explicit CLI input.
		b, err := os.ReadFile(fileOverride)
		if err != nil {
			return nil, "", err
		}
		b = bytes.TrimSpace(b)
		if len(b) == 0 {
			return nil, "", errors.New("empty key file")
		}
		return b, "file:" + fileOverride, nil
	}
	if envOverride != "" {
		if v, ok := os.LookupEnv(envOverride); ok && v != "" {
			return []byte(v), "env:" + envOverride, nil
		}
		return nil, "", fmt.Errorf("environment variable %s is not set", envOverride)
	}
	if cfg.Key.File != "" {
		// #nosec G304 -- key file path comes from the loaded config.
		b, err := os.ReadFile(cfg.Key.File)
		if err != nil {
			return nil, "", err
		}
		b = bytes.TrimSpace(b)
		if len(b) == 0 {
			return nil, "", errors.New("empty key file")
		}
		return b, "file:" + cfg.Key.File, nil
	}
	if v, ok := os.LookupEnv(cfg.Key.Env); ok && v != "" {
		return []byte(v), "env:" + cfg.Key.Env, nil
	}
	return nil, "", fmt.Errorf("no key found; set %s or configure key.file", cfg.Key.Env)
}
