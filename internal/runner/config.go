package runner

import (
	"errors"
	"fmt"
	"strings"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

const (
	DefaultSharedDir   = "/shared"
	DefaultRunnerPath  = "/shared/runner"
	DefaultConfigPath  = "/shared/vuhive.yaml"
	DefaultSummaryPath = "/shared/summary.json"
	DefaultLogPath     = "/shared/run.log"
)

// InitConfig holds configuration for the runner pod init container.
type InitConfig struct {
	SharedDir            string
	BinaryKey            string
	ConfigKey            string
	WrapperSourcePath    string
	EntrypointSourcePath string
	S3Config             s3.Config
}

// Validate checks that required fields for runner-init are provided.
func (c *InitConfig) Validate() error {
	c.SharedDir = strings.TrimSpace(c.SharedDir)
	if c.SharedDir == "" {
		c.SharedDir = DefaultSharedDir
	}

	c.BinaryKey = strings.TrimSpace(c.BinaryKey)
	if c.BinaryKey == "" {
		return fmt.Errorf("%w: binary key is required", model.ErrValidation)
	}

	c.ConfigKey = strings.TrimSpace(c.ConfigKey)
	c.WrapperSourcePath = strings.TrimSpace(c.WrapperSourcePath)
	c.EntrypointSourcePath = strings.TrimSpace(c.EntrypointSourcePath)

	return nil
}

// WrapperConfig holds configuration for the runner wrapper execution.
type WrapperConfig struct {
	RunnerPath     string
	ConfigPath     string
	SummaryPath    string
	LogPath        string
	RunID          string
	ReportKey      string
	LogsKey        string
	APICallbackURL string
	S3Config       s3.Config
}

// Validate checks that required fields for runner-wrapper are provided and sets defaults.
func (c *WrapperConfig) Validate() error {
	c.RunnerPath = strings.TrimSpace(c.RunnerPath)
	if c.RunnerPath == "" {
		c.RunnerPath = DefaultRunnerPath
	}

	c.ConfigPath = strings.TrimSpace(c.ConfigPath)
	if c.ConfigPath == "" {
		c.ConfigPath = DefaultConfigPath
	}

	c.SummaryPath = strings.TrimSpace(c.SummaryPath)
	if c.SummaryPath == "" {
		c.SummaryPath = DefaultSummaryPath
	}

	c.LogPath = strings.TrimSpace(c.LogPath)
	if c.LogPath == "" {
		c.LogPath = DefaultLogPath
	}

	c.RunID = strings.TrimSpace(c.RunID)
	c.ReportKey = strings.TrimSpace(c.ReportKey)
	c.LogsKey = strings.TrimSpace(c.LogsKey)

	// If RunID is provided and keys are empty, use standard S3 key conventions
	if c.RunID != "" {
		if c.ReportKey == "" {
			reportKey, err := s3.KeySummaryReport(c.RunID)
			if err != nil {
				return err
			}
			c.ReportKey = reportKey
		}
		if c.LogsKey == "" {
			logsKey, err := s3.KeyExecutionLogs(c.RunID)
			if err != nil {
				return err
			}
			c.LogsKey = logsKey
		}
	}

	if c.ReportKey == "" {
		return errors.New("report key is required (provide ReportKey or RunID)")
	}
	if c.LogsKey == "" {
		return errors.New("logs key is required (provide LogsKey or RunID)")
	}

	return nil
}
