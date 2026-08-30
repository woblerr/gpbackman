#!/usr/bin/env bash
set -Eeuo pipefail

TEST_COMMAND=${1:-}
HOME_DIR="/home/gpadmin"
SCRIPTS_DIR="${HOME_DIR}/run_tests"

# shellcheck source=/dev/null
source "${HOME_DIR}/e2e_databases.sh"

command_requires_standby() {
    case "${TEST_COMMAND}" in
        backup-delete|backup-clean|history-clean|history-sync)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

wait_for_service() {
    local max_attempts=${1:-10}

    for i in $(seq 1 ${max_attempts}); do
        if psql -d "${E2E_PRIMARY_DB}" -t -c "SELECT 1;" >/dev/null 2>&1; then
            echo "[INFO] Cluster ready"
            return 0
        fi
        echo "[INFO] Waiting cluster startup (${i}/${max_attempts})"
        sleep 10
    done
    echo "[ERROR] Cluster failed to start within timeout"
    return 1
}

wait_for_standby_replication() {
    local max_attempts=${1:-10}
    local state=""

    for i in $(seq 1 ${max_attempts}); do
        if state=$(psql -d "${E2E_PRIMARY_DB}" -X -A -t -c "SELECT state FROM pg_stat_replication ORDER BY CASE WHEN application_name LIKE '%walreceiver%' THEN 0 ELSE 1 END LIMIT 1;" 2>/dev/null | xargs); then
            if [ "${state}" = "streaming" ] || [ "${state}" = "catchup" ]; then
                echo "[INFO] Standby replication state: ${state}"
                return 0
            fi
        fi
        echo "[INFO] Waiting standby replication (${i}/${max_attempts})"
        sleep 10
    done
    echo "[ERROR] Standby replication did not become streaming or catchup within timeout"
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
        history-sync)
            "${SCRIPTS_DIR}/run_history-sync.sh"
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
if command_requires_standby; then
    wait_for_standby_replication
fi

echo "[INFO] Bootstrap E2E databases"
"${HOME_DIR}/bootstrap_e2e_databases.sh"

echo "[INFO] Prepare Greenplum backups"
"${HOME_DIR}/prepare_gpdb_backups.sh"

echo "[INFO] Run e2e tests for command: ${TEST_COMMAND}"
exec_test_for_command
