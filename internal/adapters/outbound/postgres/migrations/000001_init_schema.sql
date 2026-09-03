-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS test_suites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    state VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id UUID NOT NULL REFERENCES test_suites(id) ON DELETE CASCADE,
    platform VARCHAR(32) NOT NULL,
    s3_binary_key VARCHAR(512) NOT NULL DEFAULT '',
    sha256_checksum VARCHAR(64) NOT NULL DEFAULT '',
    build_logs_s3_key VARCHAR(512) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_artifacts_suite_id ON artifacts(suite_id);

CREATE TABLE IF NOT EXISTS configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id UUID NOT NULL REFERENCES test_suites(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    content_yaml TEXT NOT NULL,
    s3_config_key VARCHAR(512) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_suite_config_name UNIQUE(suite_id, name)
);

CREATE INDEX IF NOT EXISTS idx_configurations_suite_id ON configurations(suite_id);

CREATE TABLE IF NOT EXISTS runner_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    runner_image VARCHAR(256) NOT NULL DEFAULT 'alpine:3.20',
    cpu_request VARCHAR(32) NOT NULL DEFAULT '1000m',
    cpu_limit VARCHAR(32) NOT NULL DEFAULT '2000m',
    memory_request VARCHAR(32) NOT NULL DEFAULT '1Gi',
    memory_limit VARCHAR(32) NOT NULL DEFAULT '2Gi',
    node_selector JSONB NOT NULL DEFAULT '{}'::jsonb,
    affinity JSONB NOT NULL DEFAULT '{}'::jsonb,
    tolerations JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id UUID NOT NULL REFERENCES test_suites(id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    configuration_id UUID REFERENCES configurations(id) ON DELETE SET NULL,
    runner_profile_id UUID NOT NULL REFERENCES runner_profiles(id) ON DELETE RESTRICT,
    name VARCHAR(128) NOT NULL,
    cron_expression VARCHAR(64) NOT NULL,
    k8s_cronjob_name VARCHAR(128) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schedules_suite_id ON schedules(suite_id);
CREATE INDEX IF NOT EXISTS idx_schedules_is_active ON schedules(is_active);

CREATE TABLE IF NOT EXISTS test_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id UUID NOT NULL REFERENCES test_suites(id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    configuration_id UUID REFERENCES configurations(id) ON DELETE SET NULL,
    runner_profile_id UUID NOT NULL REFERENCES runner_profiles(id) ON DELETE RESTRICT,
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'QUEUED',
    k8s_job_name VARCHAR(128) NOT NULL DEFAULT '',
    k8s_namespace VARCHAR(64) NOT NULL DEFAULT 'vuhive-runners',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    exit_code INT,
    sla_passed BOOLEAN,
    total_iterations BIGINT NOT NULL DEFAULT 0,
    total_requests BIGINT NOT NULL DEFAULT 0,
    avg_tps NUMERIC(10, 2) NOT NULL DEFAULT 0,
    p50_duration_ms NUMERIC(10, 2) NOT NULL DEFAULT 0,
    p90_duration_ms NUMERIC(10, 2) NOT NULL DEFAULT 0,
    p95_duration_ms NUMERIC(10, 2) NOT NULL DEFAULT 0,
    p99_duration_ms NUMERIC(10, 2) NOT NULL DEFAULT 0,
    error_rate_pct NUMERIC(6, 3) NOT NULL DEFAULT 0,
    s3_report_key VARCHAR(512) NOT NULL DEFAULT '',
    s3_logs_key VARCHAR(512) NOT NULL DEFAULT '' ,
    summary_json JSONB,
    abort_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_test_runs_suite_id ON test_runs(suite_id);
CREATE INDEX IF NOT EXISTS idx_test_runs_status ON test_runs(status);
CREATE INDEX IF NOT EXISTS idx_test_runs_created_at ON test_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_test_runs_suite_status ON test_runs(suite_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS test_runs;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS runner_profiles;
DROP TABLE IF EXISTS configurations;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS test_suites;
-- +goose StatementEnd
