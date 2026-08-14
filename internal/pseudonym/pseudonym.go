// Package pseudonym generates deterministic pseudonyms for sensitive values.
package pseudonym

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

type Generator struct {
	key    []byte
	scope  string
	length int
}

func New(key []byte, scope string, length int) *Generator {
	return &Generator{key: key, scope: scope, length: length}
}

func (g *Generator) Token(entityType, value string) string {
	normalized := normalize(entityType, value)
	mac := hmac.New(sha256.New, g.key)
	mac.Write([]byte("nopii:v1\x00"))
	mac.Write([]byte(g.scope))
	mac.Write([]byte{0})
	mac.Write([]byte(strings.ToUpper(entityType)))
	mac.Write([]byte{0})
	mac.Write([]byte(normalized))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(mac.Sum(nil))
	if g.length > len(encoded) {
		return encoded
	}
	return encoded[:g.length]
}

func (g *Generator) Replacement(entityType, value string) string {
	return strings.ToUpper(entityType) + "_" + g.Token(entityType, value)
}

func normalize(entityType, value string) string {
	v := strings.TrimSpace(value)
	switch strings.ToUpper(entityType) {
	case "EMAIL":
		return strings.ToLower(v)
	case "UUID":
		return strings.ToLower(v)
	default:
		return v
	}
}
