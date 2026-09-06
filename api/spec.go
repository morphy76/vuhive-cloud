package api

import (
	"embed"
)

// SpecFS embeds the raw OpenAPI 3.1 specification files (both YAML and JSON).
//
//go:embed openapi.yaml openapi.json
var SpecFS embed.FS

// OpenAPISpecYAML contains the raw bytes of the OpenAPI 3.1 YAML definition.
//
//go:embed openapi.yaml
var OpenAPISpecYAML []byte

// OpenAPISpecJSON contains the raw bytes of the OpenAPI 3.1 JSON definition.
//
//go:embed openapi.json
var OpenAPISpecJSON []byte
