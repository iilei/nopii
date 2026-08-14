package stream

import (
	"bytes"
	"fmt"
	"io"

	"github.com/iilei/nopii/internal/pseudonym"
	"github.com/iilei/nopii/internal/recognizer"
)

const (
	GitMagic    = "NOPII_GIT_V1"
	GitPrettyV1 = "format:" + GitMagic + "%x1f%H%x1f%P%x1f%an%x1f%ae%x1f%cn%x1f%ce%x1f%at%x1f%ct%x1f%B%x00"
)

type Processor struct {
	gen *pseudonym.Generator
	rec *recognizer.Engine
}

func New(gen *pseudonym.Generator, rec *recognizer.Engine) *Processor {
	return &Processor{gen: gen, rec: rec}
}

func (p *Processor) Process(r io.Reader, w io.Writer) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if bytes.HasPrefix(b, []byte(GitMagic+"\x1f")) {
		return p.processGit(b, w)
	}
	_, err = io.WriteString(w, p.rec.ScrubString(string(b)))
	return err
}

func (p *Processor) processGit(b []byte, w io.Writer) error {
	records := bytes.SplitSeq(b, []byte{0})
	for rec := range records {
		rec = bytes.TrimLeft(rec, "\r\n")
		if len(bytes.TrimSpace(rec)) == 0 {
			continue
		}
		fields := bytes.SplitN(rec, []byte{0x1f}, 10)
		if len(fields) != 10 || string(fields[0]) != GitMagic {
			return fmt.Errorf("invalid %s record: expected 10 fields, got %d", GitMagic, len(fields))
		}
		hash, parents := string(fields[1]), string(fields[2])
		authorName, authorEmail := string(fields[3]), string(fields[4])
		committerName, committerEmail := string(fields[5]), string(fields[6])
		authorTime, commitTime, body := string(fields[7]), string(fields[8]), string(fields[9])
		fmt.Fprintf(w, "commit %s\n", hash)
		if parents != "" {
			fmt.Fprintf(w, "parents %s\n", parents)
		}
		fmt.Fprintf(
			w,
			"Author: %s <%s>\n",
			p.gen.Replacement("PERSON", authorName),
			p.gen.Replacement("EMAIL", authorEmail),
		)
		fmt.Fprintf(
			w,
			"Committer: %s <%s>\n",
			p.gen.Replacement("PERSON", committerName),
			p.gen.Replacement("EMAIL", committerEmail),
		)
		fmt.Fprintf(w, "AuthorDate: %s\nCommitDate: %s\n\n", authorTime, commitTime)
		fmt.Fprint(w, p.rec.ScrubString(body))
		if len(body) == 0 || body[len(body)-1] != '\n' {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}
	return nil
}
