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

const FileName = ".nopiirc.toml"

type Config struct {
	Version     int
	Scope       string
	Key         KeyConfig
	Output      OutputConfig
	Recognizers RecognizersConfig
}

type (
	KeyConfig         struct{ Env, File string }
	OutputConfig      struct{ TokenLength int }
	RecognizersConfig struct{ Email, IPv4, UUID, Phone bool }
)

func Defaults() Config {
	return Config{
		Version:     1,
		Scope:       "default",
		Key:         KeyConfig{Env: "NOPII_KEY"},
		Output:      OutputConfig{TokenLength: 12},
		Recognizers: RecognizersConfig{Email: true, IPv4: true, UUID: true, Phone: true},
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
	var path string
	var err error
	if explicit != "" {
		path, err = filepath.Abs(explicit)
		if err != nil {
			return cfg, "", err
		}
		if _, err = os.Stat(path); err != nil {
			return cfg, "", fmt.Errorf("config: %w", err)
		}
	} else {
		cwd, e := os.Getwd()
		if e != nil {
			return cfg, "", e
		}
		path, err = Discover(cwd)
		if err != nil {
			return cfg, "", err
		}
	}
	if path != "" {
		if err := parseFile(path, &cfg); err != nil {
			return cfg, path, err
		}
	}
	applyEnv(&cfg)
	if cfg.Version != 1 {
		return cfg, path, fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	if cfg.Output.TokenLength < 6 || cfg.Output.TokenLength > 52 {
		return cfg, path, errors.New("output.token_length must be between 6 and 52")
	}
	if cfg.Key.Env == "" {
		cfg.Key.Env = "NOPII_KEY"
	}
	return cfg, path, nil
}

func parseFile(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
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
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 {
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
	default:
		return fmt.Errorf("unknown config key %q", k)
	}
}

func applyEnv(c *Config) {
	if v := os.Getenv("NOPII_SCOPE"); v != "" {
		c.Scope = v
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
