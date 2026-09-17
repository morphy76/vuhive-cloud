package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// placeholderPattern matches ${secrets.KEY_NAME} where KEY_NAME is one or more uppercase letters,
// digits, or underscores, starting with an uppercase letter.
var placeholderPattern = regexp.MustCompile(`\$\{secrets\.([A-Z][A-Z0-9_]*)\}`)

// ExtractPlaceholderKeys scans a configuration template string and returns a deduplicated,
// sorted slice of secret key names referenced via ${secrets.KEY} placeholders.
// Returns nil if no placeholders are found.
func ExtractPlaceholderKeys(content string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	var keys []string
	for _, match := range matches {
		key := match[1]
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	return keys
}

// ResolveTemplatePlaceholders replaces all ${secrets.KEY} placeholders in the content
// with the corresponding values from the secrets map. Placeholders without a matching
// entry in the map are left untouched.
func ResolveTemplatePlaceholders(content string, secrets map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := placeholderPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		key := sub[1]
		if val, ok := secrets[key]; ok {
			return val
		}
		return match
	})
}

// ValidatePlaceholders extracts placeholder keys from the content and checks that all
// of them exist in the availableKeys set. Returns a slice of missing key names and
// an ErrMissingSecret error if any are absent. Returns nil, nil when all keys are satisfied.
func ValidatePlaceholders(content string, availableKeys map[string]struct{}) ([]string, error) {
	keys := ExtractPlaceholderKeys(content)
	if len(keys) == 0 {
		return nil, nil
	}

	var missing []string
	for _, key := range keys {
		if _, ok := availableKeys[key]; !ok {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return missing, fmt.Errorf("%w: %s", model.ErrMissingSecret, strings.Join(missing, ", "))
	}
	return nil, nil
}
