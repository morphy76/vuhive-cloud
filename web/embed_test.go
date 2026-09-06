package web_test

import (
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/morphy76/vuhive-cloud/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFS_EmbeddedFiles(t *testing.T) {
	distFS, err := web.GetFS()
	require.NoError(t, err)
	require.NotNil(t, distFS)

	expectedRootFiles := []string{
		"index.html",
		"manifest.webmanifest",
		"manifest.json",
		"sw.js",
		"favicon.svg",
	}

	for _, filename := range expectedRootFiles {
		t.Run("embeds root "+filename, func(t *testing.T) {
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

	t.Run("embeds hashed assets and vendor chunks", func(t *testing.T) {
		entries, err := fs.ReadDir(distFS, "assets")
		require.NoError(t, err)
		require.NotEmpty(t, entries, "assets directory must not be empty")

		var foundJS, foundCSS, foundVendorReact, foundVendorUI bool

		for _, entry := range entries {
			assert.False(t, entry.IsDir())
			name := entry.Name()

			info, err := entry.Info()
			require.NoError(t, err)
			assert.Greater(t, info.Size(), int64(0))

			if strings.HasSuffix(name, ".js") {
				foundJS = true
				if strings.HasPrefix(name, "vendor-react-") {
					foundVendorReact = true
				}
				if strings.HasPrefix(name, "vendor-ui-") {
					foundVendorUI = true
				}
			}
			if strings.HasSuffix(name, ".css") {
				foundCSS = true
			}
		}

		assert.True(t, foundJS, "expected compiled JavaScript bundle in assets/")
		assert.True(t, foundCSS, "expected compiled Tailwind CSS bundle in assets/")
		assert.True(t, foundVendorReact, "expected vendor-react chunk in assets/")
		assert.True(t, foundVendorUI, "expected vendor-ui chunk in assets/")
	})

	t.Run("fails on non-existent file", func(t *testing.T) {
		_, err := distFS.Open("nonexistent.txt")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
}
