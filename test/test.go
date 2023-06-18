// Package test provides shared test utilities
package test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type SourceProvider interface{ GetJSON() ([]byte, error) }

type SrcMake func() []byte

var _ SourceProvider = SrcMake(nil)

func (s SrcMake) GetJSON() ([]byte, error) { return s(), nil }

type SrcFile string

var _ SourceProvider = SrcFile("")

func (s SrcFile) GetJSON() ([]byte, error) {
	p := filepath.Join("..", "testdata", string(s))
	switch {
	case strings.HasSuffix(string(s), ".json"):
		return os.ReadFile(p)
	case strings.HasSuffix(string(s), ".gz"):
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("opening archive file: %w", err)
		}
		r, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("initializing gzip reader: %w", err)
		}
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("reading from gzip reader: %w", err)
		}
		return b, nil
	}
	return nil, fmt.Errorf("unsupported file: %q", s)
}

func Repeat(s string, n int) string {
	var b bytes.Buffer
	b.Grow(len(s) * n)
	for i := 0; i < n; i++ {
		b.WriteString(s)
	}
	return b.String()
}
