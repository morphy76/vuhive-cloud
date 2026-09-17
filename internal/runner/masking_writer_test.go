package runner_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskingWriter_SingleSecret(t *testing.T) {
	var buf bytes.Buffer
	w := runner.NewMaskingWriter(&buf, []string{"supersecret"})

	n, err := w.Write([]byte("authorization: Bearer supersecret in request"))
	require.NoError(t, err)
	assert.Equal(t, len("authorization: Bearer supersecret in request"), n)
	assert.Equal(t, "authorization: Bearer [REDACTED] in request", buf.String())
}

func TestMaskingWriter_MultipleSecrets(t *testing.T) {
	var buf bytes.Buffer
	w := runner.NewMaskingWriter(&buf, []string{"pass123", "tok456"})

	n, err := w.Write([]byte("connecting with pass123 and token tok456"))
	require.NoError(t, err)
	assert.Equal(t, len("connecting with pass123 and token tok456"), n)
	assert.Equal(t, "connecting with [REDACTED] and token [REDACTED]", buf.String())
}

func TestMaskingWriter_NoSecretsProducesOriginalOutput(t *testing.T) {
	input := "regular log message without secrets 123"

	// Test with nil slice
	var buf1 bytes.Buffer
	w1 := runner.NewMaskingWriter(&buf1, nil)
	n1, err1 := w1.Write([]byte(input))
	require.NoError(t, err1)
	assert.Equal(t, len(input), n1)
	assert.Equal(t, input, buf1.String())

	// Test with empty slice
	var buf2 bytes.Buffer
	w2 := runner.NewMaskingWriter(&buf2, []string{})
	n2, err2 := w2.Write([]byte(input))
	require.NoError(t, err2)
	assert.Equal(t, len(input), n2)
	assert.Equal(t, input, buf2.String())
}

func TestMaskingWriter_EmptyValuesSkipped(t *testing.T) {
	var buf bytes.Buffer
	w := runner.NewMaskingWriter(&buf, []string{"", "secret", "   ", ""})

	input := "hello secret world"
	n, err := w.Write([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, len(input), n)
	assert.Equal(t, "hello [REDACTED] world", buf.String())
}

func TestMaskingWriter_AllEmptySecrets(t *testing.T) {
	var buf bytes.Buffer
	w := runner.NewMaskingWriter(&buf, []string{"", "   ", ""})

	input := "hello world without secrets"
	n, err := w.Write([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, len(input), n)
	assert.Equal(t, input, buf.String())
}

func TestMaskingWriter_MultipleOccurrences(t *testing.T) {
	var buf bytes.Buffer
	w := runner.NewMaskingWriter(&buf, []string{"secret"})

	input := "secret before secret middle secret after"
	n, err := w.Write([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, len(input), n)
	assert.Equal(t, "[REDACTED] before [REDACTED] middle [REDACTED] after", buf.String())
}

func TestMaskingWriter_OverlappingSecrets(t *testing.T) {
	var buf bytes.Buffer
	// "secret" is a substring of "super_secret" - longer should be matched first
	w := runner.NewMaskingWriter(&buf, []string{"secret", "super_secret"})

	input := "value is super_secret and normal secret"
	n, err := w.Write([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, len(input), n)
	assert.Equal(t, "value is [REDACTED] and normal [REDACTED]", buf.String())
}

type errorTestWriter struct{}

func (e *errorTestWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated disk full")
}

func TestMaskingWriter_UnderlyingWriterError(t *testing.T) {
	w := runner.NewMaskingWriter(&errorTestWriter{}, []string{"secret"})

	_, err := w.Write([]byte("test with secret"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated disk full")
}
