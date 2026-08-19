// Package config loads and validates nopii configuration from files and environment variables.
package config

import (
	"encoding"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	FileName = ".nopiirc.toml"

	defaultConfigVersion      = 1
	defaultScope              = "default"
	defaultPseudonymAlgorithm = "v1"
	defaultKeyEnv             = "NOPII_KEY"
	defaultTokenLength        = 12
	defaultGranularitySeconds = 86400
	minTokenLength            = 6
	maxTokenLength            = 52
	minClampGranularity       = 1
	envKeyParts               = 2
	customPatternPrefix       = "NOPII_CUSTOM_PATTERN__"
)

var _ encoding.TextUnmarshaler = (*ClassifierConfig)(nil)

type (
	ClassifierConfig struct {
		Label   string `toml:"label"`
		Pattern string `toml:"pattern"`
	}

	Config struct {
		Version        int                         `toml:"version"`
		Scope          string                      `toml:"scope"`
		Pseudonyms     PseudonymConfig             `toml:"pseudonyms"`
		Key            KeyConfig                   `toml:"key"`
		Output         OutputConfig                `toml:"output"`
		Recognizers    RecognizersConfig           `toml:"recognizers"`
		Classifiers    map[string]ClassifierConfig `toml:"classifiers"`
		CustomPatterns map[string]string           `toml:"custom_patterns"`
		Git            GitConfig                   `toml:"git"`
	}
	PseudonymConfig struct {
		Algorithm string `toml:"algorithm"`
	}
	KeyConfig struct {
		Env  string `toml:"env"`
		File string `toml:"file"`
	}
	OutputConfig struct {
		TokenLength int `toml:"token_length"`
	}
	RecognizersConfig struct {
		Email bool `toml:"email"`
		IPv4  bool `toml:"ipv4"`
		UUID  bool `toml:"uuid"`
		Phone bool `toml:"phone"`
	}
	GitConfig struct {
		DateClamp DateClampConfig `toml:"date_clamp"`
	}
	DateClampConfig struct {
		Enabled            bool `toml:"enabled"`
		GranularitySeconds int  `toml:"granularity_seconds"`
	}
)

func (c *ClassifierConfig) UnmarshalText(text []byte) error {
	c.Label = string(text)
	c.Pattern = ""
	return nil
}

func Defaults() Config {
	return Config{
		Version:        defaultConfigVersion,
		Scope:          defaultScope,
		Pseudonyms:     PseudonymConfig{Algorithm: defaultPseudonymAlgorithm},
		Key:            KeyConfig{Env: defaultKeyEnv},
		Output:         OutputConfig{TokenLength: defaultTokenLength},
		Recognizers:    RecognizersConfig{Email: true, IPv4: true, UUID: true, Phone: true},
		Classifiers:    map[string]ClassifierConfig{},
		CustomPatterns: map[string]string{},
		Git: GitConfig{DateClamp: DateClampConfig{
			Enabled:            false,
			GranularitySeconds: defaultGranularitySeconds,
		}},
	}
}

func Discover(start string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	startAbs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	homeAbs, err := filepath.Abs(home)
	if err != nil {
		return "", err
	}
	cur := startAbs
	for {
		candidate := filepath.Join(cur, FileName)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
		if samePath(cur, homeAbs) {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		if isWithin(startAbs, homeAbs) && !isWithin(parent, homeAbs) {
			break
		}
		cur = parent
	}
	return "", nil
}

func Load(explicit string) (Config, string, error) {
	cfg := Defaults()
	path, err := resolveConfigPath(explicit)
	if err != nil {
		return cfg, "", err
	}
	if path != "" {
		if err := parseFile(path, &cfg); err != nil {
			return cfg, path, err
		}
	}
	normalizeClassifierNames(&cfg)
	ApplyEnv(&cfg)
	if err := validateConfig(&cfg); err != nil {
		return cfg, path, err
	}
	if cfg.Key.Env == "" {
		cfg.Key.Env = defaultKeyEnv
	}
	return cfg, path, nil
}

func resolveConfigPath(explicit string) (string, error) {
	if explicit != "" {
		path, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		if _, err = os.Stat(path); err != nil {
			return "", fmt.Errorf("config: %w", err)
		}
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Discover(cwd)
}

func validateConfig(cfg *Config) error {
	if cfg.Version != defaultConfigVersion {
		return fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	if cfg.Output.TokenLength < minTokenLength || cfg.Output.TokenLength > maxTokenLength {
		return errors.New("output.token_length must be between 6 and 52")
	}
	if cfg.Pseudonyms.Algorithm != defaultPseudonymAlgorithm {
		return fmt.Errorf("unsupported pseudonyms.algorithm %q", cfg.Pseudonyms.Algorithm)
	}
	if cfg.Git.DateClamp.Enabled && cfg.Git.DateClamp.GranularitySeconds < minClampGranularity {
		return errors.New("git.date_clamp.granularity_seconds must be at least 1")
	}
	return nil
}

func parseFile(path string, cfg *Config) error {
	// #nosec G304 -- the config path is explicitly requested by the user or discovered from the working tree.
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	decoder := toml.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func normalizeClassifierNames(c *Config) {
	if len(c.Classifiers) == 0 {
		return
	}
	normalized := make(map[string]ClassifierConfig, len(c.Classifiers))
	for name, classifier := range c.Classifiers {
		normalized[strings.ToLower(strings.TrimSpace(name))] = classifier
	}
	c.Classifiers = normalized
}

func ApplyEnv(c *Config) {
	applyEnv(c)
}

func applyEnv(c *Config) {
	applyStringEnv(&c.Scope, "NOPII_SCOPE")
	applyStringEnv(&c.Pseudonyms.Algorithm, "NOPII_PSEUDONYMS_ALGORITHM")
	applyBoolEnv(&c.Git.DateClamp.Enabled, "NOPII_GIT_DATE_CLAMP_ENABLED")
	applyIntEnv(&c.Git.DateClamp.GranularitySeconds, "NOPII_GIT_DATE_CLAMP_GRANULARITY")
	applyStringEnv(&c.Key.Env, "NOPII_KEY_ENV")
	applyStringEnv(&c.Key.File, "NOPII_KEY_FILE")
	applyIntEnv(&c.Output.TokenLength, "NOPII_TOKEN_LENGTH")
	applyCustomPatternEnv(c)
	applyClassifierFallbacks(c)
}

func applyStringEnv(dst *string, key string) {
	if v := os.Getenv(key); v != "" {
		*dst = v
	}
}

func applyBoolEnv(dst *bool, key string) {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			*dst = b
		}
	}
}

func applyIntEnv(dst *int, key string) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*dst = n
		}
	}
}

func applyCustomPatternEnv(c *Config) {
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", envKeyParts)
		if len(parts) != envKeyParts {
			continue
		}
		key, value := parts[0], parts[1]
		if !strings.HasPrefix(key, customPatternPrefix) {
			continue
		}
		if c.CustomPatterns == nil {
			c.CustomPatterns = map[string]string{}
		}
		name := strings.ToLower(strings.TrimPrefix(key, customPatternPrefix))
		c.CustomPatterns[name] = value
	}
}

func applyClassifierFallbacks(c *Config) {
	for name, classifier := range c.Classifiers {
		if classifier.Label == "" {
			classifier.Label = strings.ToUpper(name)
		}
		if pattern, ok := c.CustomPatterns[name]; ok && classifier.Pattern == "" {
			classifier.Pattern = pattern
		}
		c.Classifiers[name] = classifier
	}
}

func samePath(a, b string) bool { return filepath.Clean(a) == filepath.Clean(b) }
func isWithin(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
