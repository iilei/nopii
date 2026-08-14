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
)

type (
	Config struct {
		Version     int
		Scope       string
		Key         KeyConfig
		Output      OutputConfig
		Recognizers RecognizersConfig
		Git         GitConfig
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
		Version:     defaultConfigVersion,
		Scope:       defaultScope,
		Key:         KeyConfig{Env: defaultKeyEnv},
		Output:      OutputConfig{TokenLength: defaultTokenLength},
		Recognizers: RecognizersConfig{Email: true, IPv4: true, UUID: true, Phone: true},
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
	applyEnv(&cfg)
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
		return fmt.Errorf("unknown config key %q", k)
	}
}

func applyEnv(c *Config) {
	if v := os.Getenv("NOPII_SCOPE"); v != "" {
		c.Scope = v
	}
	if v := os.Getenv("NOPII_GIT_DATE_CLAMP_ENABLED"); v != "" {
		if b, e := parseBool(v); e == nil {
			c.Git.DateClamp.Enabled = b
		}
	}
	if v := os.Getenv("NOPII_GIT_DATE_CLAMP_GRANULARITY"); v != "" {
		if n, e := parseInt(v); e == nil {
			c.Git.DateClamp.GranularitySeconds = n
		}
	}
	if v := os.Getenv("NOPII_KEY_ENV"); v != "" {
		c.Key.Env = v
	}
	if v := os.Getenv("NOPII_KEY_FILE"); v != "" {
		c.Key.File = v
	}
	if v := os.Getenv("NOPII_TOKEN_LENGTH"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Output.TokenLength = n
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
