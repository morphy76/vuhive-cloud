package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/service"
)

func TestExtractPlaceholderKeys(t *testing.T) {
	t.Run("extracts single placeholder", func(t *testing.T) {
		yaml := `
host: example.com
password: ${secrets.DB_PASSWORD}
`
		keys := service.ExtractPlaceholderKeys(yaml)
		assert.Equal(t, []string{"DB_PASSWORD"}, keys)
	})

	t.Run("extracts multiple distinct placeholders", func(t *testing.T) {
		yaml := `
db_host: ${secrets.DB_HOST}
db_password: ${secrets.DB_PASSWORD}
api_key: ${secrets.API_KEY}
`
		keys := service.ExtractPlaceholderKeys(yaml)
		assert.ElementsMatch(t, []string{"DB_HOST", "DB_PASSWORD", "API_KEY"}, keys)
	})

	t.Run("deduplicates repeated placeholders", func(t *testing.T) {
		yaml := `
primary: ${secrets.DB_PASSWORD}
replica: ${secrets.DB_PASSWORD}
`
		keys := service.ExtractPlaceholderKeys(yaml)
		assert.Equal(t, []string{"DB_PASSWORD"}, keys)
	})

	t.Run("returns nil for no placeholders", func(t *testing.T) {
		yaml := `
host: example.com
port: 5432
`
		keys := service.ExtractPlaceholderKeys(yaml)
		assert.Nil(t, keys)
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		keys := service.ExtractPlaceholderKeys("")
		assert.Nil(t, keys)
	})

	t.Run("ignores malformed placeholders", func(t *testing.T) {
		yaml := `
good: ${secrets.VALID_KEY}
bad1: ${secret.NO_S}
bad2: $secrets.MISSING_BRACES
bad3: ${MISSING_PREFIX}
bad4: ${secrets.}
`
		keys := service.ExtractPlaceholderKeys(yaml)
		assert.Equal(t, []string{"VALID_KEY"}, keys)
	})
}

func TestResolveTemplatePlaceholders(t *testing.T) {
	t.Run("resolves single placeholder", func(t *testing.T) {
		tmpl := `password: ${secrets.DB_PASSWORD}`
		secrets := map[string]string{"DB_PASSWORD": "s3cr3t"}
		result := service.ResolveTemplatePlaceholders(tmpl, secrets)
		assert.Equal(t, `password: s3cr3t`, result)
	})

	t.Run("resolves multiple placeholders", func(t *testing.T) {
		tmpl := `host: ${secrets.DB_HOST}
password: ${secrets.DB_PASSWORD}`
		secrets := map[string]string{
			"DB_HOST":     "db.example.com",
			"DB_PASSWORD": "s3cr3t",
		}
		result := service.ResolveTemplatePlaceholders(tmpl, secrets)
		assert.Equal(t, `host: db.example.com
password: s3cr3t`, result)
	})

	t.Run("leaves unresolved placeholders untouched", func(t *testing.T) {
		tmpl := `password: ${secrets.MISSING_KEY}`
		secrets := map[string]string{}
		result := service.ResolveTemplatePlaceholders(tmpl, secrets)
		assert.Equal(t, `password: ${secrets.MISSING_KEY}`, result)
	})

	t.Run("returns original when no placeholders", func(t *testing.T) {
		tmpl := `host: example.com`
		result := service.ResolveTemplatePlaceholders(tmpl, map[string]string{})
		assert.Equal(t, `host: example.com`, result)
	})
}

func TestValidatePlaceholders(t *testing.T) {
	t.Run("all keys available returns nil", func(t *testing.T) {
		content := `
password: ${secrets.DB_PASSWORD}
key: ${secrets.API_KEY}
`
		availableKeys := map[string]struct{}{
			"DB_PASSWORD": {},
			"API_KEY":     {},
		}
		missing, err := service.ValidatePlaceholders(content, availableKeys)
		require.NoError(t, err)
		assert.Empty(t, missing)
	})

	t.Run("missing keys returns error", func(t *testing.T) {
		content := `
password: ${secrets.DB_PASSWORD}
key: ${secrets.API_KEY}
token: ${secrets.MISSING_TOKEN}
`
		availableKeys := map[string]struct{}{
			"DB_PASSWORD": {},
		}
		missing, err := service.ValidatePlaceholders(content, availableKeys)
		require.Error(t, err)
		assert.ElementsMatch(t, []string{"API_KEY", "MISSING_TOKEN"}, missing)
	})

	t.Run("no placeholders returns nil", func(t *testing.T) {
		content := `host: example.com`
		missing, err := service.ValidatePlaceholders(content, map[string]struct{}{})
		require.NoError(t, err)
		assert.Empty(t, missing)
	})
}
