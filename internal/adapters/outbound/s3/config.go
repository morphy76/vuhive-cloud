package s3

import (
	"errors"
	"strings"
)

// Config holds S3 / MinIO connection and operational parameters.
type Config struct {
	Endpoint        string // Custom endpoint URL (e.g. "http://localhost:9000" for MinIO)
	Region          string // AWS region, defaults to "us-east-1"
	Bucket          string // Target bucket name
	AccessKeyID     string // AWS Access Key
	SecretAccessKey string // AWS Secret Access Key
	UsePathStyle    bool   // Force path-style addressing (required for MinIO)
}

// Validate checks that required fields are present and applies reasonable defaults.
func (c *Config) Validate() error {
	c.Bucket = strings.TrimSpace(c.Bucket)
	if c.Bucket == "" {
		return errors.New("s3 bucket is required")
	}

	c.AccessKeyID = strings.TrimSpace(c.AccessKeyID)
	if c.AccessKeyID == "" {
		return errors.New("s3 access key id is required")
	}

	c.SecretAccessKey = strings.TrimSpace(c.SecretAccessKey)
	if c.SecretAccessKey == "" {
		return errors.New("s3 secret access key is required")
	}

	c.Region = strings.TrimSpace(c.Region)
	if c.Region == "" {
		c.Region = "us-east-1"
	}

	// For MinIO or non-AWS endpoints, path-style is generally standard
	if !c.UsePathStyle && c.Endpoint != "" {
		c.UsePathStyle = true
	}

	if c.Region == "us-east-1" && !c.UsePathStyle {
		c.UsePathStyle = true
	}

	return nil
}
