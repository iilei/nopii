// Package config loads and validates nopii configuration from files and environment variables.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	FileName = ".nopiirc.toml"

	defaultConfigVersion      = 1
	defaultScope              = "default"
	defaultKeyEnv             = "NOPII_KEY"
	defaultTokenLength        = 12
	defaultGranularitySeconds = 86400
	minTokenLength            = 6
	maxTokenLength            = 52
	minClampGranularity       = 1
	configKeyParts            = 2
	envKeyParts               = 2
	customPatternPrefix       = "NOPII_CUSTOM_PATTERN__"
)

type (
	Config struct {
		Version        int
		Scope          string
		Key            KeyConfig
		Output         OutputConfig
		Recognizers    RecognizersConfig
		Classifiers    map[string]string
		CustomPatterns map[string]string
		Git            GitConfig
	}
	KeyConfig struct {
		Env  string
		File string
	}
	OutputConfig struct {
		TokenLength int
	}
	RecognizersConfig struct {
		Email bool
		IPv4  bool
		UUID  bool
		Phone bool
	}
	GitConfig struct {
		DateClamp DateClampConfig
	}
	DateClampConfig struct {
		Enabled            bool
		GranularitySeconds int
	}
)

func Defaults() Config {
	return Config{
		Version:        defaultConfigVersion,
		Scope:          defaultScope,
		Key:            KeyConfig{Env: defaultKeyEnv},
		Output:         OutputConfig{TokenLength: defaultTokenLength},
		Recognizers:    RecognizersConfig{Email: true, IPv4: true, UUID: true, Phone: true},
		Classifiers:    map[string]string{},
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
	section := ""
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := strings.TrimSpace(stripComment(scanner.Text()))
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
			section = strings.TrimSpace(raw[1 : len(raw)-1])
			continue
		}
		parts := strings.SplitN(raw, "=", configKeyParts)
		if len(parts) != configKeyParts {
			return fmt.Errorf("%s:%d: expected key = value", path, lineNo)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		full := key
		if section != "" {
			full = section + "." + key
		}
		if err := setValue(cfg, full, val); err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
	}
	return scanner.Err()
}

func stripComment(s string) string {
	inQuote := false
	esc := false
	for i, r := range s {
		if r == '\\' && inQuote && !esc {
			esc = true
			continue
		}
		if r == '"' && !esc {
			inQuote = !inQuote
		}
		if r == '#' && !inQuote {
			return s[:i]
		}
		esc = false
	}
	return s
}

func parseString(v string) (string, error) {
	u, err := strconv.Unquote(v)
	if err != nil {
		return "", fmt.Errorf("expected quoted string, got %q", v)
	}
	return u, nil
}

func parseBool(v string) (bool, error) {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("expected boolean, got %q", v)
	}
	return b, nil
}

func parseInt(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", v)
	}
	return n, nil
}

func setValue(c *Config, k, v string) error {
	switch k {
	case "version":
		n, e := parseInt(v)
		c.Version = n
		return e
	case "scope":
		s, e := parseString(v)
		c.Scope = s
		return e
	case "key.env":
		s, e := parseString(v)
		c.Key.Env = s
		return e
	case "key.file":
		s, e := parseString(v)
		c.Key.File = s
		return e
	case "output.token_length":
		n, e := parseInt(v)
		c.Output.TokenLength = n
		return e
	case "recognizers.email":
		b, e := parseBool(v)
		c.Recognizers.Email = b
		return e
	case "recognizers.ipv4":
		b, e := parseBool(v)
		c.Recognizers.IPv4 = b
		return e
	case "recognizers.uuid":
		b, e := parseBool(v)
		c.Recognizers.UUID = b
		return e
	case "recognizers.phone":
		b, e := parseBool(v)
		c.Recognizers.Phone = b
		return e
	case "git.date_clamp.enabled":
		b, e := parseBool(v)
		c.Git.DateClamp.Enabled = b
		return e
	case "git.date_clamp.granularity_seconds":
		n, e := parseInt(v)
		c.Git.DateClamp.GranularitySeconds = n
		return e
	default:
		if strings.HasPrefix(k, "classifiers.") {
			if c.Classifiers == nil {
				c.Classifiers = map[string]string{}
			}
			name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(k, "classifiers.")))
			s, e := parseString(v)
			c.Classifiers[name] = s
			return e
		}
		return fmt.Errorf("unknown config key %q", k)
	}
}

func ApplyEnv(c *Config) {
	applyEnv(c)
}

func applyEnv(c *Config) {
	applyStringEnv(&c.Scope, "NOPII_SCOPE")
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
		if b, err := parseBool(v); err == nil {
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
	if c.CustomPatterns == nil {
		return
	}
	for name, mapping := range c.Classifiers {
		if _, ok := c.CustomPatterns[name]; !ok {
			continue
		}
		if mapping == "" {
			c.Classifiers[name] = strings.ToUpper(name)
		}
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
