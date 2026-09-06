package web_test

import (
	"io"
	"io/fs"
	"testing"

	"github.com/morphy76/vuhive-cloud/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFS_EmbeddedFiles(t *testing.T) {
	distFS, err := web.GetFS()
	require.NoError(t, err)
	require.NotNil(t, distFS)

	expectedFiles := []string{
		"index.html",
		"manifest.webmanifest",
		"manifest.json",
		"sw.js",
		"favicon.svg",
		"assets/index.js",
		"assets/index.css",
	}

	for _, filename := range expectedFiles {
		t.Run("embeds "+filename, func(t *testing.T) {
			f, err := distFS.Open(filename)
			require.NoError(t, err, "failed to open %s", filename)
			defer func() { _ = f.Close() }()

			stat, err := f.Stat()
			require.NoError(t, err)
			assert.False(t, stat.IsDir())
			assert.Greater(t, stat.Size(), int64(0))

			content, err := io.ReadAll(f)
			require.NoError(t, err)
			assert.NotEmpty(t, content)
		})
	}

	t.Run("fails on non-existent file", func(t *testing.T) {
		_, err := distFS.Open("nonexistent.txt")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
}
