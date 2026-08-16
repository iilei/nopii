package stream

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
	"github.com/iilei/nopii/internal/recognizer"
)

const (
	GitMagic = "NOPII_GIT_V1"
	// GitPrettyV1 keeps the commit message body as the final field in the v1 record.
	// That invariant lets the parser treat the last field as the body without extra
	// payload markers, while still keeping all metadata fields structured and explicit.
	GitPrettyV1 = "format:" + GitMagic + "%x1f%H%x1f%P%x1f\"%an\" <%ae>%x1f\"%cn\" <%ce>%x1f%at%x1f%ct%x1f%B%x00"

	gitRecordFieldCount = 8
)

type (
	Processor struct {
		gen   *pseudonym.Generator
		rec   *recognizer.Engine
		clamp config.DateClampConfig
	}

	// gitRecord holds the parsed fields of a single nopii-v1 git record.
	gitRecord struct {
		hash, parents                 string
		authorName, authorEmail       string
		committerName, committerEmail string
		authorTime, commitTime        string
		body                          string
	}
)

func New(gen *pseudonym.Generator, rec *recognizer.Engine, clamp config.DateClampConfig) *Processor {
	return &Processor{gen: gen, rec: rec, clamp: clamp}
}

func (p *Processor) Process(r io.Reader, w io.Writer) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if isGitStream(b) {
		return p.processGit(b, w)
	}
	_, err = io.WriteString(w, p.rec.ScrubString(string(b)))
	return err
}

// isGitStream returns true when b contains a nopii-v1 git record. It scans
// past any annotation lines a signing backend may inject before the magic.
func isGitStream(b []byte) bool {
	return bytes.Contains(b, []byte(GitMagic+"\x1f"))
}

func (p *Processor) processGit(b []byte, w io.Writer) error {
	records := bytes.SplitSeq(b, []byte{0})
	for rec := range records {
		rec = bytes.TrimLeft(rec, "\r\n")
		if len(bytes.TrimSpace(rec)) == 0 {
			continue
		}
		// Scrub and emit any annotation lines (from any signing backend)
		// that precede the nopii magic in this record.
		rec, err := p.flushAnnotationLines(rec, w)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(rec)) == 0 {
			continue
		}
		if err := p.writeRecord(rec, w); err != nil {
			return err
		}
	}
	return nil
}

func parseGitRecord(rec []byte) (gitRecord, error) {
	fields := bytes.SplitN(rec, []byte{0x1f}, gitRecordFieldCount)
	if len(fields) != gitRecordFieldCount || string(fields[0]) != GitMagic {
		return gitRecord{}, fmt.Errorf(
			"invalid %s record: expected %d fields, got %d",
			GitMagic, gitRecordFieldCount, len(fields),
		)
	}
	authorName, authorEmail, err := parseMailIdentity(string(fields[3]))
	if err != nil {
		return gitRecord{}, fmt.Errorf("invalid author identity %q: %w", fields[3], err)
	}
	committerName, committerEmail, err := parseMailIdentity(string(fields[4]))
	if err != nil {
		return gitRecord{}, fmt.Errorf("invalid committer identity %q: %w", fields[4], err)
	}
	return gitRecord{
		hash:           string(fields[1]),
		parents:        string(fields[2]),
		authorName:     authorName,
		authorEmail:    authorEmail,
		committerName:  committerName,
		committerEmail: committerEmail,
		authorTime:     string(fields[5]),
		commitTime:     string(fields[6]),
		body:           string(fields[7]),
	}, nil
}

func parseMailIdentity(v string) (string, string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", "", errors.New("empty identity")
	}
	start := strings.Index(v, " <")
	if start <= 0 || !strings.HasSuffix(v, ">") {
		return "", "", errors.New("expected mailbox form \"Name\" <email@example.com>")
	}
	name := strings.TrimSpace(v[:start])
	email := strings.TrimSpace(v[start+2 : len(v)-1])
	name = strings.Trim(name, "\"")
	if name == "" || email == "" {
		return "", "", errors.New("expected mailbox form \"Name\" <email@example.com>")
	}
	return name, email, nil
}

func (p *Processor) writeRecord(rec []byte, w io.Writer) error {
	r, err := parseGitRecord(rec)
	if err != nil {
		return err
	}
	r.authorTime = p.applyClamp(r.authorTime)
	r.commitTime = p.applyClamp(r.commitTime)
	if _, err := fmt.Fprintf(w, "commit %s\n", r.hash); err != nil {
		return err
	}
	if r.parents != "" {
		if _, err := fmt.Fprintf(w, "parents %s\n", r.parents); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(
		w,
		"Author: \"%s\" <%s>\n",
		p.gen.Replacement("PERSON", r.authorName),
		p.gen.Replacement("EMAIL", r.authorEmail),
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		w,
		"Committer: \"%s\" <%s>\n",
		p.gen.Replacement("PERSON", r.committerName),
		p.gen.Replacement("EMAIL", r.committerEmail),
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "AuthorDate: %s\nCommitDate: %s\n\n", r.authorTime, r.commitTime); err != nil {
		return err
	}
	// The v1 record always stores the raw commit body as the final field. That makes
	// the body-scope explicit without extra marker bytes, and keeps trailer matching
	// focused on the Git message payload instead of the metadata fields.
	if _, err := fmt.Fprint(w, p.rec.ScrubString(r.body)); err != nil {
		return err
	}
	if r.body == "" || r.body[len(r.body)-1] != '\n' {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(w)
	return err
}

// flushAnnotationLines scrubs and emits any lines that precede the nopii
// magic in rec, then returns rec trimmed to start at the magic. This handles
// all signing backends (GPG, SSH, X.509) without enumerating their output
// prefixes — whatever appears before NOPII_GIT_V1 is treated as annotation.
func (p *Processor) flushAnnotationLines(rec []byte, w io.Writer) ([]byte, error) {
	magic := []byte(GitMagic + "\x1f")
	idx := bytes.Index(rec, magic)
	if idx <= 0 {
		return rec, nil
	}
	annotation := rec[:idx]
	for line := range bytes.SplitSeq(bytes.TrimRight(annotation, "\r\n"), []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if _, err := fmt.Fprintln(w, p.rec.ScrubString(string(line))); err != nil {
			return rec, err
		}
	}
	return rec[idx:], nil
}

func (p *Processor) applyClamp(tsStr string) string {
	if !p.clamp.Enabled {
		return tsStr
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return tsStr
	}
	return strconv.FormatInt(ClampTimestamp(ts, p.clamp), 10)
}
