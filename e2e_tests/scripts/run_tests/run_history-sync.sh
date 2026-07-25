#!/usr/bin/env bash
set -Eeuo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="history-sync"

# Seed the standby through an explicit sync, mutate primary history without its
# automatic sync, then synchronize the now-stale standby through auto-load.
test_history_sync_after_disabled_backup_delete() {
    local timestamp
    local output
    local primary_row
    local standby_row

    output=$(run_gpbackman_capture "${COMMAND}" "seed_standby_explicit" --history-db "${HISTORY_DB}")
    assert_history_sync_success_output "${output}"

    timestamp="$(get_active_local_backup_timestamp "full")"
    output=$(run_gpbackman_capture "backup-delete" "delete_without_standby_sync" --history-db "${HISTORY_DB}" --timestamp "${timestamp}" --no-history-sync-standby)
    assert_history_sync_disabled_output "${output}"
    assert_primary_backup_deleted "${timestamp}"

    primary_row="$(primary_backup_row_for_timestamp "${timestamp}")"
    standby_row="$(standby_backup_row_for_timestamp "${timestamp}")"
    if [ -z "${primary_row}" ] || [ -z "${standby_row}" ]; then
        echo "[ERROR] Expected primary and standby history rows for ${timestamp}"
        exit 1
    fi
    if [ "${primary_row}" = "${standby_row}" ]; then
        echo "[ERROR] Standby history row should remain stale after disabled automatic sync"
        exit 1
    fi

    output=$(
        export MASTER_DATA_DIRECTORY="${DATA_DIR}"
        unset COORDINATOR_DATA_DIRECTORY
        run_gpbackman_capture "${COMMAND}" "sync_stale_standby_auto_load" --auto-load-history-db
    )
    assert_history_sync_success_output "${output}"
    assert_primary_standby_backup_row_match "${timestamp}"
}

run_test "${COMMAND}" 1 test_history_sync_after_disabled_backup_delete

log_all_tests_passed "${COMMAND}"
