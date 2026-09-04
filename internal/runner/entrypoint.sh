#!/bin/sh
set -e

# ==============================================================================
# vuhive-cloud: Runner Container Injected Entrypoint Script
# Executes the test binary, captures run.log, traps exit signals, and
# ensures report upload upon termination even on non-zero exit codes.
# ==============================================================================

# If compiled runner-wrapper binary exists, delegate directly to it
if [ -x "/shared/runner-wrapper" ]; then
    exec /shared/runner-wrapper "$@"
fi

RUNNER_BIN="${RUNNER_PATH:-/shared/runner}"
SUMMARY_FILE="${SUMMARY_PATH:-/shared/summary.json}"
LOG_FILE="${LOG_PATH:-/shared/run.log}"
CONFIG_FILE="${CONFIG_PATH:-/shared/vuhive.yaml}"

cleanup() {
    EXIT_CODE=$?
    echo "Entrypoint: runner finished with exit code ${EXIT_CODE}"
    
    # Guarantee summary.json exists
    if [ ! -s "${SUMMARY_FILE}" ]; then
        echo "Entrypoint: creating fallback summary report..."
        cat <<EOF > "${SUMMARY_FILE}"
{
  "status": "FAILED",
  "exit_code": ${EXIT_CODE},
  "error": "runner terminated with code ${EXIT_CODE} without generating summary",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "sla_passed": false
}
EOF
    fi

    # Upload to S3 if aws CLI is available and S3 keys are set
    if command -v aws >/dev/null 2>&1; then
        if [ -n "${S3_BUCKET}" ]; then
            ENDPOINT_FLAG=""
            if [ -n "${S3_ENDPOINT}" ]; then
                ENDPOINT_FLAG="--endpoint-url ${S3_ENDPOINT}"
            fi

            if [ -n "${S3_REPORT_KEY}" ] && [ -f "${SUMMARY_FILE}" ]; then
                echo "Uploading summary report to s3://${S3_BUCKET}/${S3_REPORT_KEY}..."
                aws s3 cp "${SUMMARY_FILE}" "s3://${S3_BUCKET}/${S3_REPORT_KEY}" ${ENDPOINT_FLAG} || true
            fi

            if [ -n "${S3_LOGS_KEY}" ] && [ -f "${LOG_FILE}" ]; then
                echo "Uploading execution log to s3://${S3_BUCKET}/${S3_LOGS_KEY}..."
                aws s3 cp "${LOG_FILE}" "s3://${S3_BUCKET}/${S3_LOGS_KEY}" ${ENDPOINT_FLAG} || true
            fi
        fi
    fi

    # Callback notification if URL provided
    CALLBACK_URL="${API_CALLBACK_URL:-${VUHIVE_API_URL}}"
    if [ -n "${CALLBACK_URL}" ] && command -v curl >/dev/null 2>&1; then
        echo "Sending completion callback to ${CALLBACK_URL}..."
        curl -s -X POST "${CALLBACK_URL}" \
            -H "Content-Type: application/json" \
            -d "{\"run_id\":\"${VUHIVE_RUN_ID:-${RUN_ID}}\",\"exit_code\":${EXIT_CODE},\"report_key\":\"${S3_REPORT_KEY}\",\"logs_key\":\"${S3_LOGS_KEY}\"}" || true
    fi

    exit ${EXIT_CODE}
}

trap cleanup EXIT INT TERM

ARGS="--summary-export=${SUMMARY_FILE}"
if [ -f "${CONFIG_FILE}" ]; then
    ARGS="${ARGS} --config=${CONFIG_FILE}"
fi

echo "Entrypoint: executing ${RUNNER_BIN} ${ARGS} $@..."
"${RUNNER_BIN}" ${ARGS} "$@" 2>&1 | tee "${LOG_FILE}"
