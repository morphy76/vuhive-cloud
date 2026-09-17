package runner

import (
	"io"
	"sort"
	"strings"
)

// MaskingWriter wraps an io.Writer and replaces occurrences of secret values with [REDACTED].
type MaskingWriter struct {
	w        io.Writer
	replacer *strings.Replacer
}

// Compile-time interface assertion
var _ io.Writer = (*MaskingWriter)(nil)

// NewMaskingWriter creates a new MaskingWriter wrapping w that masks all non-empty secret values
// with [REDACTED] using strings.NewReplacer.
func NewMaskingWriter(w io.Writer, secretValues []string) *MaskingWriter {
	seen := make(map[string]struct{})
	var uniqueSecrets []string
	for _, s := range secretValues {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			uniqueSecrets = append(uniqueSecrets, s)
		}
	}

	if len(uniqueSecrets) == 0 {
		return &MaskingWriter{
			w:        w,
			replacer: nil,
		}
	}

	// Sort secrets by length descending so that longer matching secrets are replaced first
	// (e.g. "super_secret" before "secret").
	sort.Slice(uniqueSecrets, func(i, j int) bool {
		return len(uniqueSecrets[i]) > len(uniqueSecrets[j])
	})

	pairs := make([]string, 0, len(uniqueSecrets)*2)
	for _, s := range uniqueSecrets {
		pairs = append(pairs, s, "[REDACTED]")
	}

	return &MaskingWriter{
		w:        w,
		replacer: strings.NewReplacer(pairs...),
	}
}

// Write writes p to the underlying writer with all secret values masked as [REDACTED].
// Returns len(p) on successful write to conform to the io.Writer contract expected by callers like io.Copy.
func (m *MaskingWriter) Write(p []byte) (int, error) {
	if m.replacer == nil {
		return m.w.Write(p)
	}

	masked := m.replacer.Replace(string(p))
	_, err := m.w.Write([]byte(masked))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
