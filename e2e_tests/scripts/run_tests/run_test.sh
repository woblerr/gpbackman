#!/usr/bin/env bash
set -Eeuo pipefail

TEST_COMMAND=${1:-}
GP_DB_NAME="demo"
HOME_DIR="/home/gpadmin"
SCRIPTS_DIR="${HOME_DIR}/run_tests"

wait_for_service() {
    local max_attempts=${1:-10}

    for i in $(seq 1 ${max_attempts}); do
        if psql -d ${GP_DB_NAME} -t -c  "SELECT 1;" >/dev/null 2>&1; then
            echo "[INFO] Cluster ready"
            return 0
        fi
        echo "[INFO] Waiting cluster startup (${i}/${max_attempts})"
        sleep 10
    done
    echo "[ERROR] Cluster failed to start within timeout"
    return 1
}


exec_test_for_command() {
    case "${TEST_COMMAND}" in
        backup-info)
            "${SCRIPTS_DIR}/run_backup-info.sh"
            ;;
        report-info)
            "${SCRIPTS_DIR}/run_report-info.sh"
            ;;
        backup-delete)
            "${SCRIPTS_DIR}/run_backup-delete.sh"
            ;;
        backup-clean)
            "${SCRIPTS_DIR}/run_backup-clean.sh"
            ;;
        history-clean)
            "${SCRIPTS_DIR}/run_history-clean.sh"
            ;;
        history-migrate)
            "${SCRIPTS_DIR}/run_history-migrate.sh"
            ;;
        *)
            echo "[ERROR] Unknown test command: ${TEST_COMMAND}"
            exit 1
            ;;
    esac
}

echo "[INFO] Check Greenplum cluster"
sleep 90
wait_for_service

echo "[INFO] Prepare Greenplum backups"
"${HOME_DIR}/prepare_gpdb_backups.sh"

echo "[INFO] Run e2e tests for command: ${TEST_COMMAND}"
exec_test_for_command